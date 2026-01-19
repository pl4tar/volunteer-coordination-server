package main

import (
	"context"
	"log/slog"
	"os/signal"
	"syscall"
	"volunteer-coordination-server/config"
	_ "volunteer-coordination-server/docs"
	router "volunteer-coordination-server/internal/handler/http"
	"volunteer-coordination-server/internal/storage"

	"github.com/gin-gonic/gin"
)

// @title Volunteer Coordination Server API
// @version 1.0
// @description API для системы координации волонтёров.
// @description Конфиг поднимается из CONFIG_PATH (например: ./config/local.yaml).
// @description DB: host=localhost port=5432 db=volonteer user=postgres (пароль берётся из переменных окружения).
//
// @host localhost:4949
// @BasePath /api/v1
//
// @schemes http
//
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
//
// @tag.name auth
// @tag.description Регистрация/логин/refresh
//
// @tag.name users
// @tag.description Пользователи и профиль

func main() {
	cfg := config.GetConfig()
	gin.ForceConsoleColor()
	slog.Info("Starting app")
	slog.Debug("Debud messages are enabled")
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	cfg.Client = storage.NewConnection(ctx, cfg)
	r := router.SetupRouter(ctx, cfg, cfg.Client)
	err := r.Run(":" + cfg.Env.API_PORT)
	if err != nil {
		slog.Error("server start", "error", err)
	}
}
