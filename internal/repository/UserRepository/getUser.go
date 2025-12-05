package UserRepository

import (
	"context"
	"time"
	"volunteer-coordination-server/internal/entity"

	"github.com/google/uuid"
)

// QueryGetUserByEmail - получает пользователя из бд по email
func (r *UsrRepository) QueryGetUserByEmail(ctx context.Context, email string) (entity.User, error) {
	var (
		ID        uuid.UUID
		Email     string
		FullName  string
		password  string
		createdAt time.Time
	)

	err := r.db.QueryRow(ctx,
		`SELECT id, email, full_name, role FROM users
		WHERE email=$1`,
		email).Scan(
		&ID,
		&Email,
		&FullName,
		&password,
		&createdAt,
	)
	if err != nil {
		return entity.User{}, err
	}
	return entity.User{
		ID:           ID,
		Email:        Email,
		PasswordHash: password,
		FullName:     FullName,
		CreatedAt:    createdAt,
	}, nil
}

// QueryGetUserByID - получает пользователя из бд по uuid v4, но id передается как string
func (r *UsrRepository) QueryGetUserByID(ctx context.Context, id string) (entity.User, error) {
	var (
		ID        uuid.UUID
		Email     string
		FullName  string
		password  string
		createdAt time.Time
	)
	err := r.db.QueryRow(ctx,
		`SELECT id, email, full_name, password_hash, created_at FROM users
	WHERE id=$1`,
		id).Scan(
		&ID,
		&Email,
		&FullName,
		&password,
		&createdAt,
	)
	if err != nil {
		return entity.User{}, err
	}
	return entity.User{
		ID:           ID,
		Email:        Email,
		PasswordHash: password,
		FullName:     FullName,
		CreatedAt:    createdAt,
	}, nil
}
