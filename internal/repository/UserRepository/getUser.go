package repository

import (
	"context"
	"fmt"
	"log/slog"
	"volunteer-coordination-server/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// GetUserByEmail - получает пользователя из бд по email
func (r *UserRepository) QueryGetUserByEmail(ctx context.Context, email string) (domain.User, error) {
	const op = "repository.UserRepository.GetUserByEmail"

	query := `
		SELECT 
			id, 
			email, 
			password_hash, 
			full_name, 
			phone, 
			created_at, 
			updated_at, 
			is_blocked
		FROM users 
		WHERE email = $1
	`

	var user domain.User

	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.FullName,
		&user.Phone,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.IsBlocked,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			slog.Warn("user not found by email",
				slog.String("op", op),
				slog.String("email", email))
			return domain.User{}, fmt.Errorf("%s: user not found", op)
		}

		slog.Error("failed to get user by email",
			slog.String("op", op),
			slog.String("email", email),
			slog.Any("error", err))
		return domain.User{}, fmt.Errorf("%s: %w", op, err)
	}

	// Получаем роли пользователя
	roles, err := r.GetUserRoles(ctx, user.ID)
	if err != nil {
		slog.Error("failed to get user roles",
			slog.String("op", op),
			slog.String("user_id", user.ID.String()),
			slog.Any("error", err))
		// Не прерываем выполнение, возвращаем пользователя без ролей
	} else {
		user.Roles = roles
	}

	return user, nil
}

// QueryGetUserByID - получает пользователя из бд по uuid
func (r *UserRepository) QueryGetUserByID(ctx context.Context, id uuid.UUID) (domain.User, error) {
	const op = "repository.UserRepository.GetUserByID"

	query := `
		SELECT 
			id, 
			email, 
			password_hash, 
			full_name, 
			phone, 
			created_at, 
			updated_at, 
			is_blocked
		FROM users 
		WHERE id = $1
	`

	var user domain.User

	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.FullName,
		&user.Phone,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.IsBlocked,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			slog.Warn("user not found by ID",
				slog.String("op", op),
				slog.String("user_id", id.String()))
			return domain.User{}, fmt.Errorf("%s: user not found", op)
		}

		slog.Error("failed to get user by ID",
			slog.String("op", op),
			slog.String("user_id", id.String()),
			slog.Any("error", err))
		return domain.User{}, fmt.Errorf("%s: %w", op, err)
	}

	// Получаем роли пользователя
	roles, err := r.GetUserRoles(ctx, user.ID)
	if err != nil {
		slog.Error("failed to get user roles",
			slog.String("op", op),
			slog.String("user_id", user.ID.String()),
			slog.Any("error", err))
	} else {
		user.Roles = roles
	}

	return user, nil
}

// GetUserRoles - получает роли пользователя
func (r *UserRepository) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]string, error) {
	const op = "repository.UserRepository.GetUserRoles"

	query := `
		SELECT role 
		FROM user_roles 
		WHERE user_id = $1
		ORDER BY role
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		slog.Error("failed to query user roles",
			slog.String("op", op),
			slog.String("user_id", userID.String()),
			slog.Any("error", err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var roles []string
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			slog.Error("failed to scan role",
				slog.String("op", op),
				slog.String("user_id", userID.String()),
				slog.Any("error", err))
			continue
		}
		roles = append(roles, role)
	}

	if err := rows.Err(); err != nil {
		slog.Error("rows iteration error",
			slog.String("op", op),
			slog.String("user_id", userID.String()),
			slog.Any("error", err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return roles, nil
}
