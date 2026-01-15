package main

import (
	"context"
	"log/slog"
	"os/signal"
	"syscall"
	"volunteer-coordination-server/config"
	router "volunteer-coordination-server/internal/handler/http"
	"volunteer-coordination-server/internal/storage"

	"github.com/gin-gonic/gin"
)

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
