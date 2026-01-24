package v1

import (
	"time"
	"volunteer-coordination-server/internal/domain"

	"github.com/google/uuid"
)

// User DTOs
type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	FullName  string    `json:"full_name"`
	Phone     string    `json:"phone,omitempty"`
	Roles     []string  `json:"roles"`
	IsBlocked bool      `json:"is_blocked"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type SuccessOnlyResponse struct {
	Success bool `json:"success"`
}

type RegisterRequest struct {
	Email    string   `json:"email" binding:"required,email"`
	Password string   `json:"password" binding:"required,min=6"`
	FullName string   `json:"full_name" binding:"required"`
	Phone    string   `json:"phone,omitempty"`
	Roles    []string `json:"roles,omitempty"`
}

type UpdateUserRequest struct {
	FullName string `json:"full_name,omitempty"`
	Phone    string `json:"phone,omitempty"`
	Bio      string `json:"bio,omitempty"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=6"`
}

// Auth DTOs
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}
type SessionDTO struct {
	ID        uuid.UUID `json:"id"`
	ExpiresAt time.Time `json:"expires_at"`
}

type LoginResponse struct {
	Message string       `json:"message"`
	User    UserResponse `json:"user"`
	Session SessionDTO   `json:"session"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type RefreshTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

// Events DTOs
type EventListResponse struct {
	Items []any `json:"items"`
	Total int   `json:"total"`
	Page  int   `json:"page"`
	Limit int   `json:"limit"`
}

type ReviewRequest struct {
	Rating  int16  `json:"rating" binding:"required"`
	Comment string `json:"comment" binding:"required"`
}

// Helper DTOs
type SuccessResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}

// org DTOs

type EventUpsertRequest domain.EventUpsert

type decideRegistrationRequest struct {
	Status string `json:"status" binding:"required"` // confirmed|rejected|attended|no_show
}

type notifyRequest struct {
	Type    string         `json:"type" binding:"required"`
	Payload map[string]any `json:"payload" binding:"required"`
}

type scoreRequest struct {
	Score   int    `json:"score" binding:"required"` // 0..100
	Comment string `json:"comment,omitempty"`
}

type organizerRequest struct {
	CompanyName string         `json:"company_name,omitempty"`
	Documents   map[string]any `json:"documents,omitempty"`
}

// AdminStatsResponse swagger response
type AdminStatsResponse struct {
	Success bool                   `json:"success"`
	Data    map[string]interface{} `json:"data"`
}

// AdminSettingsResponse swagger response
type AdminSettingsResponse struct {
	Success bool                   `json:"success"`
	Key     string                 `json:"key"`
	Value   map[string]interface{} `json:"value"`
}

// AdminListResponse swagger response for lists
type AdminListResponse[T any] struct {
	Success bool `json:"success"`
	Items   []T  `json:"items"`
}
