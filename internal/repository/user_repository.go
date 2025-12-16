package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"volunteer-coordination-server/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type userRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) UserRepository {
	return &userRepository{db: db}
}

// GetUserByEmail - получает пользователя из бд по email
func (r *userRepository) QueryGetUserByEmail(ctx context.Context, email string) (domain.User, error) {
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
		if errors.Is(err, pgx.ErrNoRows) {
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

// QueryGetUserByID - получает пользователя из бд по uuid
func (r *userRepository) QueryGetUserByID(ctx context.Context, id uuid.UUID) (domain.User, error) {
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
func (r *userRepository) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]domain.UserRole, error) {
	const op = "repository.UserRepository.GetUserRoles"

	query := `
		SELECT role, granted_at 
		FROM user_roles 
		WHERE user_id = $1
		ORDER BY granted_at
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

	var roles []domain.UserRole
	for rows.Next() {
		var role domain.UserRole
		if err := rows.Scan(&role.Role, &role.GrantedAt); err != nil {
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

// QueryNewUser - создает запись в бд, принимает структуру domain.UserRegistration
// Валидация на существующих пользователей реализована на стороне бд
func (r *userRepository) QueryNewUser(ctx context.Context, user domain.UserRegistration) (string, error) {
	var userID string
	PasswordHash, _ := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)

	tx, err := r.db.Begin(ctx)
	if err != nil {
		slog.Error("Error starting transaction", "error", err)
		return "", err
	}

	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		}
	}()

	err = tx.QueryRow(
		ctx,
		`SELECT create_user($1, $2, $3)`,
		user.Email,
		PasswordHash,
		user.FullName,
	).Scan(&userID)

	if err != nil {
		slog.Error("Error querying user", "error", err)
		return "", err
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO user_roles (user_id, role) VALUES ($1, $2) 
				 ON CONFLICT (user_id, role) DO NOTHING`,
		userID, user.Roles)
	if err != nil {
		slog.Error("Error inserting new user", "error", err)
		return "", err
	}

	if err = tx.Commit(ctx); err != nil {
		slog.Error("Error committing transaction", "error", err)
		return "", err
	}

	return userID, err
}
