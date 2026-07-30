package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"espx/internal/auth"
	"espx/internal/config"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := auth.Serve(ctx, cfg); err != nil && err != context.Canceled {
		slog.Error("auth server stopped", "error", err)
		os.Exit(1)
	}
}
