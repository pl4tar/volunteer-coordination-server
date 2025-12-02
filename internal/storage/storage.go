package storage

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"time"
	"volunteer-coordination-server/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewConnection(ctx context.Context, cfg *config.Config) *pgxpool.Pool {
	env := cfg.Env

	dsn := fmt.Sprintf("postgresql://%s:%s@%s:%d/%s",
		env.DB_USERNAME,
		env.DB_PASSWORD,
		env.DB_HOST,
		env.DB_PORT,
		env.DB_NAME,
	)

	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		log.Fatal("Unable to parse config", err)
	}

	// Настройки пула
	config.MaxConns = 25
	config.MinConns = 5
	config.MaxConnLifetime = 30 * time.Minute
	config.MaxConnIdleTime = 5 * time.Minute

	conn, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		slog.Error("ошибка при подключении к БД", slog.Any("error", err))
		panic(err)
	}

	if err = CheckAndMigrate(conn); err != nil {
		slog.Error("ошибка при миграции базы данных", slog.Any("error", err))
		panic(err)
	}

	slog.Info("DB connected")

	return conn
}
