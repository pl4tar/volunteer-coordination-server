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
	config      *config.Config
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
		config:      cfg,
	}
}

// Login выполняет аутентификацию пользователя
func (s *AuthService) Login(ctx context.Context, email, password, userAgent, ip string) (domain.AuthResult, error) {
	const op = "service.AuthService.Login"

	// Находим пользователя
	user, err := s.userRepo.QueryGetUserByEmail(ctx, email)
	if err != nil {
		slog.Warn("user not found",
			slog.String("op", op),
			slog.String("email", email))
		return domain.AuthResult{}, fmt.Errorf("%s: invalid credentials", op)
	}

	// Проверяем блокировку
	if user.IsBlocked {
		slog.Warn("blocked user tried to login",
			slog.String("op", op),
			slog.String("user_id", user.ID.String()))
		return domain.AuthResult{}, domain.ErrUserBlocked
	}

	// Проверяем пароль
	if err := user.CheckPassword(password); err != nil {
		slog.Warn("invalid password",
			slog.String("op", op),
			slog.String("user_id", user.ID.String()))
		return domain.AuthResult{}, domain.ErrInvalidPassword
	}

	// Создаем сессию
	session := domain.NewSession(user.ID, userAgent, ip, 30*24*time.Hour) // 30 дней
	// Сохраняем сессию
	if err := s.sessionRepo.Save(ctx, session); err != nil {
		slog.Error("failed to save session",
			slog.String("op", op),
			slog.String("user_id", user.ID.String()),
			slog.Any("error", err))
		return domain.AuthResult{}, fmt.Errorf("%s: failed to create session", op)
	}

	// Генерируем токены
	tokens, err := s.tokenSvc.GenerateTokens(user.ID.String(), user.Email, user.GetRoles())
	if err != nil {
		slog.Error("failed to generate tokens",
			slog.String("op", op),
			slog.String("user_id", user.ID.String()),
			slog.Any("error", err))
		return domain.AuthResult{}, fmt.Errorf("%s: failed to generate tokens", op)
	}

	// Обновляем refresh токен в сессии с JWT токеном
	session.RefreshToken = tokens.RefreshToken
	if err := s.sessionRepo.Save(ctx, session); err != nil {
		slog.Error("failed to update session with JWT token",
			slog.String("op", op),
			slog.String("session_id", session.ID.String()),
			slog.Any("error", err))
		// Не прерываем, пользователь уже аутентифицирован
	}

	slog.Info("user logged in successfully",
		slog.String("op", op),
		slog.String("user_id", user.ID.String()),
		slog.String("email", user.Email))

	return domain.AuthResult{
		User:    user,
		Session: session,
		Tokens:  tokens,
	}, nil
}

// RefreshTokens обновляет access токен по refresh токену
func (s *AuthService) RefreshTokens(ctx context.Context, refreshToken string) (*domain.AuthResult, error) {
	const op = "service.AuthService.RefreshTokens"

	claims, err := s.tokenSvc.ValidateRefreshToken(refreshToken)
	if err != nil {
		slog.Warn("invalid refresh token",
			slog.String("op", op),
			slog.Any("error", err))
		return nil, fmt.Errorf("%s: invalid refresh token", op)
	}

	// Находим сессию по refresh токену
	session, err := s.sessionRepo.GetSessionByRefreshToken(ctx, refreshToken)
	if err != nil {
		slog.Warn("session not found for refresh token",
			slog.String("op", op),
			slog.String("user_id", claims.Subject))
		return nil, fmt.Errorf("%s: session not found", op)
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		slog.Error("invalid user ID in token",
			slog.String("op", op),
			slog.String("subject", claims.Subject))
		return nil, fmt.Errorf("%s: invalid user ID", op)
	}

	user, err := s.userRepo.QueryGetUserByID(ctx, userID)
	if err != nil {
		slog.Error("user not found",
			slog.String("op", op),
			slog.String("user_id", userID.String()))
		return nil, fmt.Errorf("%s: user not found", op)
	}

	// Проверяем блокировку
	if user.IsBlocked {
		slog.Warn("blocked user tried to refresh tokens",
			slog.String("op", op),
			slog.String("user_id", user.ID.String()))
		return nil, domain.ErrUserBlocked
	}

	// Обновляем сессию
	session.Renew(30 * 24 * time.Hour)

	// Генерируем новые токены
	tokens, err := s.tokenSvc.GenerateTokens(user.ID.String(), user.Email, user.GetRoles())
	if err != nil {
		slog.Error("failed to generate new tokens",
			slog.String("op", op),
			slog.String("user_id", user.ID.String()),
			slog.Any("error", err))
		return nil, fmt.Errorf("%s: failed to generate tokens", op)
	}

	// Обновляем refresh токен в сессии
	session.RefreshToken = tokens.RefreshToken
	if err := s.sessionRepo.Save(ctx, session); err != nil {
		slog.Error("failed to update session",
			slog.String("op", op),
			slog.String("session_id", session.ID.String()),
			slog.Any("error", err))
		// Не прерываем, токены сгенерированы
	}

	slog.Info("tokens refreshed successfully",
		slog.String("op", op),
		slog.String("user_id", user.ID.String()),
		slog.String("session_id", session.ID.String()))

	return &domain.AuthResult{
		User:    user,
		Session: session,
		Tokens:  tokens,
	}, nil
}

// Logout выходит из текущей сессии
func (s *AuthService) Logout(ctx context.Context, sessionID uuid.UUID) error {
	const op = "service.AuthService.Logout"

	if err := s.sessionRepo.Delete(ctx, sessionID); err != nil {
		slog.Error("failed to delete session",
			slog.String("op", op),
			slog.String("session_id", sessionID.String()),
			slog.Any("error", err))
		return fmt.Errorf("%s: failed to logout", op)
	}

	slog.Info("user logged out", "session_id", sessionID)
	return nil
}

// LogoutAll выходит из всех сессий пользователя
func (s *AuthService) LogoutAll(ctx context.Context, userID uuid.UUID) error {
	const op = "service.AuthService.LogoutAll"

	if err := s.sessionRepo.DeleteByUserID(ctx, userID); err != nil {
		slog.Error("failed to delete all user sessions",
			slog.String("op", op),
			slog.String("user_id", userID.String()),
			slog.Any("error", err))
		return fmt.Errorf("%s: failed to logout from all devices", op)
	}

	slog.Info("user logged out from all devices", "user_id", userID)
	return nil
}

// GetCurrentUser получает текущего пользователя
func (s *AuthService) GetCurrentUser(ctx context.Context, userID uuid.UUID) (domain.User, error) {
	const op = "service.AuthService.GetCurrentUser"

	user, err := s.userRepo.QueryGetUserByID(ctx, userID)
	if err != nil {
		slog.Error("failed to get user",
			slog.String("op", op),
			slog.String("user_id", userID.String()),
			slog.Any("error", err))
		return domain.User{}, fmt.Errorf("%s: failed to get user", op)
	}

	return user, nil
}
