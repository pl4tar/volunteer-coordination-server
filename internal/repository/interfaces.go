package repository

import (
	"context"
	"errors"
	"volunteer-coordination-server/internal/domain"

	"github.com/google/uuid"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
)

type UserRepository interface {
	Save(ctx context.Context, user domain.User) (string, error)
	QueryGetUserByEmail(ctx context.Context, email string) (domain.User, error)
	QueryGetUserByID(ctx context.Context, userID uuid.UUID) (domain.User, error)

	ExistsByEmail(ctx context.Context, email string) (bool, error)
	ListUsers(ctx context.Context, page, limit int, roleFilter string) ([]domain.User, int64, error)

	UpdateProfile(ctx context.Context, user domain.User) error
	UpdatePassword(ctx context.Context, userID uuid.UUID, newPasswordHash string) error

	AddUserRole(ctx context.Context, userID uuid.UUID, role string) error
	RemoveUserRole(ctx context.Context, userID uuid.UUID, role string) error

	BlockUser(ctx context.Context, userID uuid.UUID) error
	UnblockUser(ctx context.Context, userID uuid.UUID) error
}

// SessionRepository интерфейс для работы с сессиями
type SessionRepository interface {
	Save(ctx context.Context, session domain.Session) error
	UpdateUserSession(ctx context.Context, refreshToken string, sessionID uuid.UUID) error
	GetSessionByRefreshToken(ctx context.Context, refreshToken string) (domain.Session, error)
	Delete(ctx context.Context, sessionID uuid.UUID) error
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error

	FindByID(ctx context.Context, id uuid.UUID) (domain.Session, error)
	DeleteExpired(ctx context.Context) error
}
