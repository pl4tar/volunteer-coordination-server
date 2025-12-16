package storage

import (
	"context"
	_ "embed"
	"fmt"
	_ "volunteer-coordination-server/internal/repository/UserRepository"

	"github.com/jackc/pgx/v5/pgxpool"
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

	//if count == 0 {
	//	if err = InsertTestUser(context.Background(), db); err != nil {
	//		return fmt.Errorf("ошибка добавления пользователя: %v", err)
	//	}
	//}

	return nil
}

//func InsertTestUser(ctx context.Context, db *pgxpool.Pool) error {
//	var adminUser = domain.UserRegistration{
//		Email:    "admin@ya.ru",
//		Password: "123456",
//		FullName: "admin",
//		Role:     "admin",
//	}
//	_, err := User.QueryNewAdminUser(ctx, db, adminUser)
//
//	var volunteerUser = domain.UserRegistration{
//		Email:    "volonteer@ya.ru",
//		Password: "987654321",
//		FullName: "Vasya Pupkin",
//		Role:     "volunteer",
//	}
//	_, err = User.QueryNewVolunteerUser(ctx, db, volunteerUser)
//
//	return err
//}
