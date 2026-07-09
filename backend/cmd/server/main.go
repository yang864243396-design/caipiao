package main

import (
	"log/slog"
	"os"

	"github.com/joho/godotenv"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/server"
)

func main() {
	_ = godotenv.Load()

	cfg := config.Load()
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	srv, err := server.New(cfg)
	if err != nil {
		slog.Error("server init failed", "err", err)
		os.Exit(1)
	}
	defer srv.Close()

	slog.Info("backend listening", "addr", ":"+cfg.Port, "env", cfg.Env, "base", "/api/v1")
	if err := srv.ListenAndServe(); err != nil {
		slog.Error("server stopped", "err", err)
		os.Exit(1)
	}
}
