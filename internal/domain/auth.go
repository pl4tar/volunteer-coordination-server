package domain

import (
	"time"

	"github.com/google/uuid"
)

// Tokens - пара access и refresh токенов
type Tokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

// Session Структура для сессии
type Session struct {
	ID           uuid.UUID `json:"id"`
	UserID       uuid.UUID `json:"user_id"`
	RefreshToken string    `json:"refresh_token"`
	UserAgent    string    `json:"user_agent,omitempty"`
	IPAddress    string    `json:"ip_address,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// AuthResult результат аутентификации
type AuthResult struct {
	User    User
	Session Session
	Tokens  *Tokens
}

// JWTClaims данные в JWT токене
type JWTClaims struct {
	UserID string   `json:"user_id"`
	Email  string   `json:"email"`
	Roles  []string `json:"roles"`
}

// NewSession создает новую сессию
func NewSession(userID uuid.UUID, userAgent, ip string, duration time.Duration) Session {
	now := time.Now()
	return Session{
		ID:           uuid.New(),
		UserID:       userID,
		RefreshToken: uuid.New().String(),
		UserAgent:    userAgent,
		IPAddress:    ip,
		CreatedAt:    now,
		ExpiresAt:    now.Add(duration),
	}
}

// IsExpired истекла ли сессия
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// Renew обновляет сессию
func (s *Session) Renew(duration time.Duration) {
	s.ExpiresAt = time.Now().Add(duration)
	s.RefreshToken = uuid.New().String()
}
