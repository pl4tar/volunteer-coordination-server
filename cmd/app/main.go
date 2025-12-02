package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os/signal"
	"syscall"
	"time"
	"volunteer-coordination-server/config"
	"volunteer-coordination-server/internal/router"
	"volunteer-coordination-server/internal/storage"
)

func main() {
	cfg := config.GetConfig()

	slog.Info("Starting app")
	slog.Debug("Debud messages are enabled")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	cfg.Client = storage.NewConnection(ctx, cfg)
	r := router.SetupRouter(ctx, cfg.Client)
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Env.API_PORT),
		Handler:      r,
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
	}

	go func() {
		slog.Info("Server run")
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("⚫️ Server %v", slog.String("error", err.Error()))
			panic(err)
		}
	}()
	<-ctx.Done()
	slog.Info("⚫️ Graceful shutdown initiated...")
	if err := server.Shutdown(ctx); err != nil {
		slog.Error("⚫️ Server forced to shutdown", slog.String("error", err.Error()))
		panic(err)
	}
}
