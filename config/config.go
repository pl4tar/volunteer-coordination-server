package config

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

type Env struct {
	CONFIG_PATH string `env:"CONFIG_PATH"`
	DB_NAME     string `env:"DB_NAME"`
	DB_USERNAME string `env:"DB_USERNAME"`
	DB_PASSWORD string `env:"DB_PASSWORD"`
	DB_PORT     int    `env:"DB_PORT"`
	DB_HOST     string `env:"DB_HOST"`
	IpAddress   string `env:"IP_ADDRESS"`
	API_PORT    string `env:"API_PORT"`
}

type HTTPConfig struct {
	Port         string        `yaml:"port" env:"HTTP_PORT"`
	ReadTimeout  time.Duration `yaml:"readTimeout" env:"HTTP_READ_TIMEOUT"`
	WriteTimeout time.Duration `yaml:"writeTimeout" env:"HTTP_WRITE_TIMEOUT"`
	Host         string        `yaml:"host" env:"HTTP_HOST"`
}

type CacheConfig struct {
	TTL time.Duration `yaml:"ttl" env:"CACHE_TTL"`
}

type JWTConfig struct {
	Secret        string `yaml:"secret" env:"JWT_SECRET"`
	RefreshSecret string `yaml:"refresh_secret" env:"JWT_REFRESH_SECRET"`
	Issuer        string `yaml:"issuer" env:"JWT_ISSUER"`

	AccessTokenTTL  string `yaml:"accessToken" env:"JWT_ACCESS_TOKEN_TTL"`
	RefreshTokenTTL string `yaml:"refreshToken" env:"JWT_REFRESH_TOKEN_TTL"`

	AccessTokenTTLParsed  time.Duration
	RefreshTokenTTLParsed time.Duration
}

type AuthConfig struct {
	UseCookies    bool   `yaml:"useCookies" env:"AUTH_USE_COOKIES"`
	CookieDomain  string `yaml:"cookieDomain" env:"AUTH_COOKIE_DOMAIN"`
	SecureCookies bool   `yaml:"secureCookies" env:"AUTH_SECURE_COOKIES"`
}

type Config struct {
	Env    Env
	HTTP   HTTPConfig
	Cache  CacheConfig
	JWT    JWTConfig
	Auth   AuthConfig
	Client *pgxpool.Pool
}

var config Config

// Load загружает конфигурацию из YAML файла и переменных окружения
func Load() (*Config, error) {
	path := os.Getenv("CONFIG_PATH")
	if path == "" {
		path = "config.yml"
	}
	var cfg Config

	loadFromYAML(path, &cfg)
	loadFromEnv(&cfg)

	accessTTL, err := time.ParseDuration(cfg.JWT.AccessTokenTTL)
	if err != nil {
		return nil, fmt.Errorf("invalid accessToken TTL: %w", err)
	}
	refreshTTL, err := time.ParseDuration(cfg.JWT.RefreshTokenTTL)
	if err != nil {
		return nil, fmt.Errorf("invalid refreshToken TTL: %w", err)
	}
	cfg.JWT.AccessTokenTTLParsed = accessTTL
	cfg.JWT.RefreshTokenTTLParsed = refreshTTL

	return &cfg, nil
}

// loadFromYAML загружает конфигурацию из YAML файла
func loadFromYAML(filename string, cfg *Config) {
	file, err := os.Open(filename)
	if err != nil {
		slog.Warn("No YAML config file found, using defaults",
			slog.String("filename", filename),
			slog.Any("error", err))
		setDefaults(cfg)
		return
	}
	defer func(file *os.File) {
		if err := file.Close(); err != nil {
			slog.Warn("Failed to close YAML config file")
		}
	}(file)

	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(cfg); err != nil {
		slog.Error("Failed to parse YAML config",
			slog.String("filename", filename),
			slog.Any("error", err))
		panic(err)
	}

	slog.Info("Configuration loaded from YAML",
		slog.String("filename", filename))
}

// loadFromEnv загружает конфигурацию из .env файла и переменных окружения
func loadFromEnv(cfg *Config) {
	if err := godotenv.Load(); err != nil {
		slog.Debug("No .env file found, using system environment variables")
	}

	var envVars Env
	if err := env.Parse(&envVars); err != nil {
		slog.Error("Failed to parse environment variables",
			slog.Any("error", err))
		panic(err)
	}
	cfg.Env = envVars

	if err := env.Parse(&cfg.HTTP); err != nil {
		slog.Warn("Failed to parse HTTP env variables, using defaults",
			slog.Any("error", err))
	}
	if err := env.Parse(&cfg.Cache); err != nil {
		slog.Warn("Failed to parse Cache env variables, using defaults",
			slog.Any("error", err))
	}
	if err := env.Parse(&cfg.JWT); err != nil {
		slog.Warn("Failed to parse JWT env variables, using defaults",
			slog.Any("error", err))
	}
	if err := env.Parse(&cfg.Auth); err != nil {
		slog.Warn("Failed to parse Auth env variables, using defaults",
			slog.Any("error", err))
	}
}

// setDefaults устанавливает значения по умолчанию
func setDefaults(cfg *Config) {
	*cfg = Config{
		HTTP: HTTPConfig{
			Port:         "4949",
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			Host:         "localhost",
		},
		Cache: CacheConfig{
			TTL: 3600 * time.Second,
		},
		JWT: JWTConfig{
			Secret:        "your-super-secret-jwt-key-change-in-production",
			RefreshSecret: "your-refresh-secret-key-change-also",
			Issuer:        "volunteer-coordination-server",

			AccessTokenTTL:  "120m",
			RefreshTokenTTL: "43200m",
		},
		Auth: AuthConfig{
			UseCookies:    false,
			CookieDomain:  "localhost",
			SecureCookies: false,
		},
	}
}
