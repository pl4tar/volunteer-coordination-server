package config

import (
	"log/slog"

	"github.com/caarlos0/env/v11"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

type Env struct {
	DB_NAME     string `env:"DB_NAME"`
	DB_USERNAME string `env:"DB_USERNAME"`
	DB_PASSWORD string `env:"DB_PASSWORD"`
	DB_PORT     int    `env:"DB_PORT"`
	DB_HOST     string `env:"DB_HOST"`
	IpAddress   string `env:"IP_ADDRESS"`
	API_PORT    string `env:"API_PORT"`
}

type Config struct {
	Env    Env
	Client *pgxpool.Pool
}

var config Config

func GetConfig() *Config {
	config.Env = *getEnv()

	return &config
}

func getEnv() *Env {
	err := godotenv.Load()
	if err != nil {
		slog.Warn("No .env file found, using system environment variables")
	}

	var cfg Env
	err = env.Parse(&cfg)
	if err != nil {
		slog.Error("Failed to parse env: %v", slog.Any("error", err))
		panic(err)
	}

	return &cfg
}
