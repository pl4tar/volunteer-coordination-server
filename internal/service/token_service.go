// internal/service/token_service.go
package service

import (
	"errors"
	"fmt"
	"log/slog"
	"time"
	"volunteer-coordination-server/config"
	"volunteer-coordination-server/internal/domain"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrTokenExpired     = errors.New("token expired")
	ErrInvalidToken     = errors.New("invalid token")
	ErrInvalidSignature = errors.New("invalid signature")
)

type TokenService struct {
	config *config.Config
}

func NewTokenService(cfg *config.Config) *TokenService {
	return &TokenService{
		config: cfg,
	}
}

// Claims для JWT токенов
type AccessTokenClaims struct {
	UserID string   `json:"user_id"`
	Email  string   `json:"email"`
	Roles  []string `json:"roles"`
	jwt.RegisteredClaims
}

// GenerateTokens создает пару access и refresh токенов
func (s *TokenService) GenerateTokens(userID, email string, roles []string) (*domain.Tokens, error) {
	const op = "service.TokenService.GenerateTokens"

	// Access Token
	accessToken, err := s.generateAccessToken(userID, email, roles)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	// Refresh Token
	refreshToken, err := s.generateRefreshToken(userID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	tokens := &domain.Tokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(s.config.JWT.AccessTokenTTLParsed.Seconds()),
	}

	slog.Debug("tokens generated", "user_id", userID)
	return tokens, nil
}

// generateAccessToken создает access token
func (s *TokenService) generateAccessToken(userID, email string, roles []string) (string, error) {
	claims := AccessTokenClaims{
		UserID: userID,
		Email:  email,
		Roles:  roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.config.JWT.AccessTokenTTLParsed)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    s.config.JWT.Issuer,
			Subject:   userID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.JWT.Secret))
}

// generateRefreshToken создает refresh token
func (s *TokenService) generateRefreshToken(userID string) (string, error) {
	claims := jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.config.JWT.RefreshTokenTTLParsed)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		Issuer:    s.config.JWT.Issuer,
		Subject:   userID,
		ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.JWT.RefreshSecret))
}

// ValidateAccessToken проверяет access token
func (s *TokenService) ValidateAccessToken(tokenString string) (*AccessTokenClaims, error) {
	const op = "service.TokenService.ValidateAccessToken"

	token, err := jwt.ParseWithClaims(tokenString, &AccessTokenClaims{},
		func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(s.config.JWT.Secret), nil
		})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if claims, ok := token.Claims.(*AccessTokenClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrInvalidToken
}

// ValidateRefreshToken проверяет refresh token
func (s *TokenService) ValidateRefreshToken(tokenString string) (*jwt.RegisteredClaims, error) {
	const op = "service.TokenService.ValidateRefreshToken"

	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{},
		func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(s.config.JWT.RefreshSecret), nil
		})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if claims, ok := token.Claims.(*jwt.RegisteredClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrInvalidToken
}

// ParseTokenWithoutValidation парсит токен без валидации (только для отладки)
func (s *TokenService) ParseTokenWithoutValidation(tokenString string) (*AccessTokenClaims, error) {
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())

	token, _, err := parser.ParseUnverified(tokenString, &AccessTokenClaims{})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*AccessTokenClaims); ok {
		return claims, nil
	}

	return nil, ErrInvalidToken
}
