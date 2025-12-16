package repository

import (
	"context"
	"volunteer-coordination-server/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

type UserRepositoryInterface interface {
	QueryNewUser(ctx context.Context, user domain.UserRegistration) (string, error)
	QueryGetUserByEmail(ctx context.Context, email string) (domain.User, error)
	QueryGetUserByID(ctx context.Context, userID uuid.UUID) (domain.User, error)
}
