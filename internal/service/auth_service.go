// internal/service/auth_service.go
package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"volunteer-coordination-server/config"
	"volunteer-coordination-server/internal/domain"
	"volunteer-coordination-server/internal/repository"

	"github.com/google/uuid"
)

type AuthService struct {
	userRepo    repository.UserRepository
	sessionRepo repository.SessionRepository
	tokenSvc    *TokenService
	cfg         *config.Config
}

func NewAuthService(
	userRepo repository.UserRepository,
	sessionRepo repository.SessionRepository,
	tokenSvc *TokenService,
	cfg *config.Config,
) *AuthService {
	return &AuthService{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		tokenSvc:    tokenSvc,
		cfg:         cfg,
	}
}

// sessionTTL — логично привязать к refresh TTL
func (s *AuthService) sessionTTL() time.Duration {
	// refresh TTL у тебя есть в cfg.JWT.RefreshTokenTTL
	return s.cfg.JWT.RefreshTokenTTLParsed
}

// Register: создаёт пользователя + сессию + токены (как “автологин”)
func (s *AuthService) Register(ctx context.Context, reg domain.UserRegistration, userAgent, ip string) (domain.AuthResult, error) {
	const op = "service.AuthService.Register"

	exists, err := s.userRepo.ExistsByEmail(ctx, reg.Email)
	if err != nil {
		return domain.AuthResult{}, fmt.Errorf("%s: %w", op, err)
	}
	if exists {
		return domain.AuthResult{}, fmt.Errorf("%s: email already registered", op)
	}

	user, err := domain.NewUser(reg)
	if err != nil {
		return domain.AuthResult{}, fmt.Errorf("%s: %w", op, err)
	}

	// ВАЖНО: сохраняем и забираем реальный ID из БД
	savedID, err := s.userRepo.Save(ctx, user)
	if err != nil {
		return domain.AuthResult{}, fmt.Errorf("%s: %w", op, err)
	}
	user.ID = savedID // <-- КЛЮЧЕВАЯ СТРОКА (фикс рассинхрона)

	// создаём сессию
	session := domain.NewSession(user.ID, userAgent, ip, s.sessionTTL())
	if err := s.sessionRepo.Save(ctx, &session); err != nil {
		return domain.AuthResult{}, fmt.Errorf("%s: failed to create session: %w", op, err)
	}

	// генерим токены ТОЛЬКО по user.ID (который уже = savedID)
	tokens, err := s.tokenSvc.GenerateTokens(user.ID.String(), user.Email, user.GetRoles())
	if err != nil {
		return domain.AuthResult{}, fmt.Errorf("%s: failed to generate tokens: %w", op, err)
	}

	// обновляем refresh токен в сессии
	session.RefreshToken = tokens.RefreshToken
	if err := s.sessionRepo.UpdateUserSession(ctx, session.RefreshToken, session.ExpiresAt, session.ID); err != nil {
		return domain.AuthResult{}, fmt.Errorf("%s: failed to update session: %w", op, err)
	}

	slog.Info("user registered",
		slog.String("op", op),
		slog.String("user_id", user.ID.String()),
		slog.String("email", user.Email),
		slog.String("session_id", session.ID.String()))

	return domain.AuthResult{User: user, Session: session, Tokens: tokens}, nil
}

// Login выполняет аутентификацию пользователя
func (s *AuthService) Login(ctx context.Context, email, password, userAgent, ip string) (domain.AuthResult, error) {
	const op = "service.AuthService.Login"

	user, err := s.userRepo.QueryGetUserByEmail(ctx, email)
	if err != nil {
		return domain.AuthResult{}, fmt.Errorf("%s: invalid credentials", op)
	}

	if user.IsBlocked {
		return domain.AuthResult{}, domain.ErrUserBlocked
	}

	if err := user.CheckPassword(password); err != nil {
		return domain.AuthResult{}, domain.ErrInvalidPassword
	}

	// создаём сессию
	session := domain.NewSession(user.ID, userAgent, ip, s.sessionTTL())

	if err := s.sessionRepo.Save(ctx, &session); err != nil {
		return domain.AuthResult{}, fmt.Errorf("%s: failed to create session: %w", op, err)
	}

	// генерим токены
	tokens, err := s.tokenSvc.GenerateTokens(user.ID.String(), user.Email, user.GetRoles())
	if err != nil {
		return domain.AuthResult{}, fmt.Errorf("%s: failed to generate tokens: %w", op, err)
	}

	// обновляем refresh токен в БД
	session.RefreshToken = tokens.RefreshToken
	if err := s.sessionRepo.UpdateUserSession(ctx, session.RefreshToken, session.ExpiresAt, session.ID); err != nil {
		return domain.AuthResult{}, fmt.Errorf("%s: failed to update session: %w", op, err)
	}

	slog.Info("user logged in",
		slog.String("op", op),
		slog.String("user_id", user.ID.String()),
		slog.String("session_id", session.ID.String()))

	return domain.AuthResult{User: user, Session: session, Tokens: tokens}, nil
}

// RefreshTokens обновляет access token по refresh токену
func (s *AuthService) RefreshTokens(ctx context.Context, refreshToken string) (*domain.AuthResult, error) {
	const op = "service.AuthService.RefreshTokens"

	claims, err := s.tokenSvc.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("%s: invalid refresh token", op)
	}

	// Проверяем, что такая сессия вообще существует и не истекла
	session, err := s.sessionRepo.GetSessionByRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, fmt.Errorf("%s: session not found", op)
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return nil, fmt.Errorf("%s: invalid user id in token", op)
	}

	user, err := s.userRepo.QueryGetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("%s: user not found", op)
	}

	if user.IsBlocked {
		return nil, domain.ErrUserBlocked
	}

	// продлеваем сессию
	session.ExpiresAt = time.Now().Add(s.sessionTTL())

	// генерим новые токены
	tokens, err := s.tokenSvc.GenerateTokens(user.ID.String(), user.Email, user.GetRoles())
	if err != nil {
		return nil, fmt.Errorf("%s: failed to generate tokens", op)
	}

	// обновляем refresh в БД на новый JWT refresh
	session.RefreshToken = tokens.RefreshToken
	if err := s.sessionRepo.UpdateUserSession(ctx, session.RefreshToken, session.ExpiresAt, session.ID); err != nil {
		return nil, fmt.Errorf("%s: failed to update session", op)
	}

	slog.Info("tokens refreshed",
		slog.String("op", op),
		slog.String("user_id", user.ID.String()),
		slog.String("session_id", session.ID.String()))

	return &domain.AuthResult{User: user, Session: session, Tokens: tokens}, nil
}

func (s *AuthService) Logout(ctx context.Context, sessionID uuid.UUID) error {
	const op = "service.AuthService.Logout"

	if err := s.sessionRepo.Delete(ctx, sessionID); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (s *AuthService) LogoutAll(ctx context.Context, userID uuid.UUID) error {
	const op = "service.AuthService.LogoutAll"

	if err := s.sessionRepo.DeleteByUserID(ctx, userID); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (s *AuthService) GetCurrentUser(ctx context.Context, userID uuid.UUID) (domain.User, error) {
	const op = "service.AuthService.GetCurrentUser"

	user, err := s.userRepo.QueryGetUserByID(ctx, userID)
	if err != nil {
		return domain.User{}, fmt.Errorf("%s: %w", op, err)
	}
	return user, nil
}
