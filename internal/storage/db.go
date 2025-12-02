package storage

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

//go:embed init.sql
var initSQL string

func CheckAndMigrate(db *pgxpool.Pool) error {
	_, err := db.Exec(context.Background(), initSQL)
	if err != nil {
		return fmt.Errorf("ошибка выполнения init.sql: %w", err)
	}

	var count int
	err = db.QueryRow(context.Background(), "select count(*) from users").Scan(&count)
	if err != nil {
		return fmt.Errorf("ошибка проверки количества пользователей в бд: %v", err)
	}

	if count == 0 {
		if err = InsertAdminUser(context.Background(), db); err != nil {
			return fmt.Errorf("ошибка добавления пользователя: %v", err)
		}
	}

	return nil
}

func InsertAdminUser(ctx context.Context, db *pgxpool.Pool) error {
	hash, _ := bcrypt.GenerateFromPassword([]byte("1111"), bcrypt.DefaultCost)

	_, err := db.Exec(
		context.Background(),
		`INSERT INTO users (email, password_hash, full_name)
		VALUES ($1, $2, $3)
		RETURNING id
		`,
		"admin@admin.ru",
		hash,
		"admin",
	)

	return err
}
