package repository

import (
	"context"
	"log/slog"
	"volunteer-coordination-server/internal/domain"

	"golang.org/x/crypto/bcrypt"
)

//Пакет реализует функции создания новых пользователей четко разделенные по ролям
//TODO доделать добавление био и телефона

// QueryNewUser - создает запись в бд, принимает структуру domain.UserRegistration
// Валидация на существующих пользователей реализована на стороне бд
func (r *UserRepository) QueryNewUser(ctx context.Context, user domain.UserRegistration) (string, error) {
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
		userID, user.Role)
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
