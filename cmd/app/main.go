package main

func main() {
	//cfg := config.GetConfig()
	//
	//slog.Info("Starting app")
	//slog.Debug("Debud messages are enabled")
	//
	//ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	//defer stop()
	//cfg.Client = storage.NewConnection(ctx, cfg)
	//server := &http.Server{
	//	Addr:         fmt.Sprintf(":%d", cfg.Env.API_PORT),
	//	Handler:      handler.Setup(cfg, ctx),
	//	WriteTimeout: time.Second * 15,
	//	ReadTimeout:  time.Second * 15,
	//	IdleTimeout:  time.Second * 60,
	//}
	//
	//go func() {
	//	slog.Info("Server run")
	//	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
	//		slog.Error("⚫️ Server %v", slog.String("error", err.Error()))
	//		panic(err)
	//	}
	//}()
	//<-ctx.Done()
	//slog.Info("⚫️ Graceful shutdown initiated...")
	//if err := server.Shutdown(ctx); err != nil {
	//	slog.Error("⚫️ Server forced to shutdown", slog.String("error", err.Error()))
	//	panic(err)
	//}
}
