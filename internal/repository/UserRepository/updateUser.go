package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

func QueryDeleteUser(ctx context.Context, db *pgxpool.Pool, mail string) error {
	_, err := db.Exec(ctx, "DELETE FROM users WHERE email = $1", mail)

	return err
}

func QueryUpdateUserPassword(ctx context.Context, db *pgxpool.Pool, userID string, newPassword string) error {
	newHashedPassword, _ := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)

	_, err := db.Exec(
		ctx,
		`UPDATE users
		SET password = $1
		WHERE id = $2
	`,
		newHashedPassword,
		userID)

	return err
}
