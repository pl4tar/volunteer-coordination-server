package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
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

func (r *userRepository) ListUsers(ctx context.Context, page, limit int, roleFilter string) ([]domain.User, int64, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}

	offset := (page - 1) * limit

	var total int64

	countSQL := `
		SELECT COUNT(DISTINCT u.id)
		FROM users u
	`
	args := []any{}
	where := []string{}

	if strings.TrimSpace(roleFilter) != "" {
		countSQL += ` JOIN user_roles ur ON ur.user_id = u.id `
		where = append(where, fmt.Sprintf("ur.role = $%d", len(args)+1))
		args = append(args, roleFilter)
	}

	if len(where) > 0 {
		countSQL += " WHERE " + strings.Join(where, " AND ")
	}

	if err := r.db.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("repository.userRepository.ListUsers count: %w", err)
	}

	selectSQL := `
		SELECT DISTINCT u.id, u.email, u.full_name, u.phone, u.password_hash, u.is_blocked, u.created_at, u.updated_at
		FROM users u
	`
	args2 := []any{}
	where2 := []string{}

	if strings.TrimSpace(roleFilter) != "" {
		selectSQL += ` JOIN user_roles ur ON ur.user_id = u.id `
		where2 = append(where2, fmt.Sprintf("ur.role = $%d", len(args2)+1))
		args2 = append(args2, roleFilter)
	}

	if len(where2) > 0 {
		selectSQL += " WHERE " + strings.Join(where2, " AND ")
	}

	// pagination
	selectSQL += fmt.Sprintf(" ORDER BY u.created_at DESC LIMIT $%d OFFSET $%d", len(args2)+1, len(args2)+2)
	args2 = append(args2, limit, offset)

	rows, err := r.db.Query(ctx, selectSQL, args2...)
	if err != nil {
		return nil, 0, fmt.Errorf("repository.userRepository.ListUsers select: %w", err)
	}
	defer rows.Close()

	users := make([]domain.User, 0, limit)

	for rows.Next() {
		u := domain.User{}
		if err := rows.Scan(
			&u.ID,
			&u.Email,
			&u.FullName,
			&u.Phone,
			&u.Password,
			&u.IsBlocked,
			&u.CreatedAt,
			&u.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("repository.userRepository.ListUsers scan: %w", err)
		}

		// если у тебя роли вытаскиваются отдельным запросом:
		roles, err := r.GetUserRoles(ctx, u.ID)
		if err != nil && err != pgx.ErrNoRows {
			return nil, 0, fmt.Errorf("repository.userRepository.ListUsers roles: %w", err)
		}
		u.Roles = roles

		users = append(users, u)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("repository.userRepository.ListUsers rows: %w", err)
	}

	return users, total, nil
}

// QueryGetUserByEmail - получает пользователя из бд по email
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

	user := domain.User{}

	var phone sql.NullString

	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.Password, // лучше назвать PasswordHash, но пока оставим как есть
		&user.FullName,
		&phone,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.IsBlocked,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			slog.Warn("user not found by email", slog.String("op", op), slog.String("email", email))
			return domain.User{}, fmt.Errorf("%s: user not found", op)
		}

		slog.Error("failed to get user by email", slog.String("op", op), slog.String("email", email), slog.Any("error", err))
		return domain.User{}, fmt.Errorf("%s: %w", op, err)
	}

	if phone.Valid {
		user.Phone = phone.String
	} else {
		user.Phone = "" // или оставь как есть, если у тебя Phone string
	}

	roles, err := r.GetUserRoles(ctx, user.ID)
	if err != nil {
		slog.Error("failed to get user roles", slog.String("op", op), slog.String("user_id", user.ID.String()), slog.Any("error", err))
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
		&user.Password,
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

// Save - создает запись в бд, принимает структуру [domain.User]
// Валидация на существующих пользователей реализована на стороне бд
func (r *userRepository) Save(ctx context.Context, user domain.User) (uuid.UUID, error) {
	var userID uuid.UUID

	// 1) password_hash генерируем в Save (но не делаем двойной bcrypt)
	passwordHash := user.Password

	// bcrypt-хеш обычно начинается с "$2a$", "$2b$", "$2y$"
	isBcryptHash := len(passwordHash) >= 4 && passwordHash[0:3] == "$2"
	if !isBcryptHash {
		// значит пришёл plaintext — хешируем
		hash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
		if err != nil {
			slog.Error("Error hashing password", "error", err)
			return uuid.Nil, err
		}
		passwordHash = string(hash)
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		slog.Error("Error starting transaction", "error", err)
		return uuid.Nil, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	// 2) создаём пользователя (через твою DB-функцию create_user)
	err = tx.QueryRow(
		ctx,
		`SELECT create_user($1, $2, $3)`,
		user.Email,
		[]byte(passwordHash), // create_user у тебя принимал bytes (как было)
		user.FullName,
	).Scan(&userID)
	if err != nil {
		slog.Error("Error querying user", "error", err)
		return uuid.Nil, err
	}

	// 3) роли вставляем по одной, а не как []domain.UserRole одним параметром
	// У тебя в БД точно есть granted_at, потому что ты его SELECT'ишь в GetUserRoles.
	// Если в БД стоит default NOW() — вставка с granted_at тоже нормально работает.
	const insertRole = `
		INSERT INTO user_roles (user_id, role, granted_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, role) DO NOTHING
	`

	now := time.Now()
	for _, role := range user.Roles {
		grantedAt := role.GrantedAt
		if grantedAt.IsZero() {
			grantedAt = now
		}

		_, err = tx.Exec(ctx, insertRole, userID, role.Role, grantedAt)
		if err != nil {
			slog.Error("Error inserting user role", "error", err)
			return uuid.Nil, err
		}
	}

	if err = tx.Commit(ctx); err != nil {
		slog.Error("Error committing transaction", "error", err)
		return uuid.Nil, err
	}

	return userID, nil
}

func (r *userRepository) UpdateProfile(ctx context.Context, user domain.User) error {
	const op = "repository.UserRepository.UpdateProfile"

	tx, err := r.db.Begin(ctx)
	if err != nil {
		slog.Error("Error starting transaction",
			slog.String("op", op),
			slog.String("user_id", user.ID.String()),
			slog.Any("error", err))
		return fmt.Errorf("%s: %w", op, err)
	}

	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
				slog.Error("Error rolling back transaction",
					slog.String("op", op),
					slog.String("user_id", user.ID.String()),
					slog.Any("rollback_error", rollbackErr))
			}
		}
	}()

	query := `
		UPDATE users SET
			email = COALESCE(NULLIF($2, ''), email),
			full_name = COALESCE(NULLIF($3, ''), full_name),
			phone = COALESCE(NULLIF($4, ''), phone),
			is_blocked = $5,
			updated_at = $6
		WHERE id = $1
	`

	result, err := tx.Exec(ctx, query,
		user.ID,
		user.Email,
		user.FullName,
		user.Phone,
		user.IsBlocked,
		user.UpdatedAt,
	)

	if err != nil {
		slog.Error("Failed to update user profile",
			slog.String("op", op),
			slog.String("user_id", user.ID.String()),
			slog.Any("error", err))
		return fmt.Errorf("%s: %w", op, err)
	}

	if result.RowsAffected() == 0 {
		slog.Warn("User not found for update",
			slog.String("op", op),
			slog.String("user_id", user.ID.String()))
		return fmt.Errorf("%s: user not found", op)
	}

	// Если переданы роли, обновляем их
	if len(user.Roles) > 0 {
		if err := r.updateUserRoles(ctx, tx, user.ID, user.Roles); err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
	}

	// Коммитим транзакцию
	if err = tx.Commit(ctx); err != nil {
		slog.Error("Failed to commit transaction",
			slog.String("op", op),
			slog.String("user_id", user.ID.String()),
			slog.Any("error", err))
		return fmt.Errorf("%s: %w", op, err)
	}

	slog.Info("User profile updated successfully",
		slog.String("op", op),
		slog.String("user_id", user.ID.String()),
		slog.String("email", user.Email))

	return nil
}

// updateUserRoles обновляет роли пользователя
func (r *userRepository) updateUserRoles(ctx context.Context, tx pgx.Tx, userID uuid.UUID, roles []domain.UserRole) error {
	const op = "repository.UserRepository.updateUserRoles"

	// Удаляем старые роли
	deleteQuery := `DELETE FROM user_roles WHERE user_id = $1`
	_, err := tx.Exec(ctx, deleteQuery, userID)
	if err != nil {
		slog.Error("Failed to delete user roles",
			slog.String("op", op),
			slog.String("user_id", userID.String()),
			slog.Any("error", err))
		return fmt.Errorf("%s: %w", op, err)
	}

	// Добавляем новые роли
	for _, role := range roles {
		insertQuery := `INSERT INTO user_roles (user_id, role, granted_at) VALUES ($1, $2, $3)`
		_, err = tx.Exec(ctx, insertQuery, userID, role.Role, role.GrantedAt)
		if err != nil {
			slog.Error("Failed to insert user role",
				slog.String("op", op),
				slog.String("user_id", userID.String()),
				slog.String("role", role.Role),
				slog.Any("error", err))
			return fmt.Errorf("%s: %w", op, err)
		}
	}

	slog.Debug("User roles updated",
		slog.String("op", op),
		slog.String("user_id", userID.String()),
		slog.Int("roles_count", len(roles)))

	return nil
}

// UpdatePassword обновляет пароль пользователя
func (r *userRepository) UpdatePassword(ctx context.Context, userID uuid.UUID, newPasswordHash string) error {
	const op = "repository.UserRepository.UpdatePassword"

	query := `
		UPDATE users SET
			password_hash = $2,
			updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.db.Exec(ctx, query, userID, newPasswordHash)
	if err != nil {
		slog.Error("Failed to update password",
			slog.String("op", op),
			slog.String("user_id", userID.String()),
			slog.Any("error", err))
		return fmt.Errorf("%s: %w", op, err)
	}

	if result.RowsAffected() == 0 {
		slog.Warn("User not found for password update",
			slog.String("op", op),
			slog.String("user_id", userID.String()))
		return fmt.Errorf("%s: user not found", op)
	}

	slog.Info("Password updated successfully",
		slog.String("op", op),
		slog.String("user_id", userID.String()))

	return nil
}

// BlockUser блокирует пользователя
func (r *userRepository) BlockUser(ctx context.Context, userID uuid.UUID) error {
	const op = "repository.UserRepository.BlockUser"

	query := `
		UPDATE users SET
			is_blocked = true,
			updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		slog.Error("Failed to block user",
			slog.String("op", op),
			slog.String("user_id", userID.String()),
			slog.Any("error", err))
		return fmt.Errorf("%s: %w", op, err)
	}

	if result.RowsAffected() == 0 {
		slog.Warn("User not found for blocking",
			slog.String("op", op),
			slog.String("user_id", userID.String()))
		return fmt.Errorf("%s: user not found", op)
	}

	slog.Info("User blocked successfully",
		slog.String("op", op),
		slog.String("user_id", userID.String()))

	return nil
}

// UnblockUser разблокирует пользователя
func (r *userRepository) UnblockUser(ctx context.Context, userID uuid.UUID) error {
	const op = "repository.UserRepository.UnblockUser"

	query := `
		UPDATE users SET
			is_blocked = false,
			updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		slog.Error("Failed to unblock user",
			slog.String("op", op),
			slog.String("user_id", userID.String()),
			slog.Any("error", err))
		return fmt.Errorf("%s: %w", op, err)
	}

	if result.RowsAffected() == 0 {
		slog.Warn("User not found for unblocking",
			slog.String("op", op),
			slog.String("user_id", userID.String()))
		return fmt.Errorf("%s: user not found", op)
	}

	slog.Info("User unblocked successfully",
		slog.String("op", op),
		slog.String("user_id", userID.String()))

	return nil
}

// AddUserRole добавляет роль пользователю
func (r *userRepository) AddUserRole(ctx context.Context, userID uuid.UUID, role string) error {
	const op = "repository.UserRepository.AddUserRole"

	query := `
		INSERT INTO user_roles (user_id, role, granted_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (user_id, role) DO UPDATE SET
			granted_at = EXCLUDED.granted_at
	`

	_, err := r.db.Exec(ctx, query, userID, role)
	if err != nil {
		slog.Error("Failed to add user role",
			slog.String("op", op),
			slog.String("user_id", userID.String()),
			slog.String("role", role),
			slog.Any("error", err))
		return fmt.Errorf("%s: %w", op, err)
	}

	slog.Info("User role added successfully",
		slog.String("op", op),
		slog.String("user_id", userID.String()),
		slog.String("role", role))

	return nil
}

// RemoveUserRole удаляет роль у пользователя
func (r *userRepository) RemoveUserRole(ctx context.Context, userID uuid.UUID, role string) error {
	const op = "repository.UserRepository.RemoveUserRole"

	query := `DELETE FROM user_roles WHERE user_id = $1 AND role = $2`

	result, err := r.db.Exec(ctx, query, userID, role)
	if err != nil {
		slog.Error("Failed to remove user role",
			slog.String("op", op),
			slog.String("user_id", userID.String()),
			slog.String("role", role),
			slog.Any("error", err))
		return fmt.Errorf("%s: %w", op, err)
	}

	if result.RowsAffected() == 0 {
		slog.Warn("User role not found for removal",
			slog.String("op", op),
			slog.String("user_id", userID.String()),
			slog.String("role", role))
		return fmt.Errorf("%s: role not found", op)
	}

	slog.Info("User role removed successfully",
		slog.String("op", op),
		slog.String("user_id", userID.String()),
		slog.String("role", role))

	return nil
}

// ExistsByEmail проверяет существование пользователя по email
func (r *userRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	const op = "postgres.userRepository.ExistsByEmail"

	query := `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`
	var exists bool

	err := r.db.QueryRow(ctx, query, email).Scan(&exists)
	if err != nil {
		slog.Error("failed to check email existence",
			slog.String("op", op),
			slog.String("email", email),
			slog.Any("error", err))
		return false, fmt.Errorf("%s: %w", op, err)
	}

	slog.Debug("email existence checked",
		slog.String("op", op),
		slog.String("email", email),
		slog.Bool("exists", exists))

	return exists, nil
}
