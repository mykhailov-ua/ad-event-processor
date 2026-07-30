package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"espx/internal/config"
	"espx/internal/management"
	"espx/pkg/runtimeautotune"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}
	runtimeautotune.Apply(cfg)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := management.Serve(ctx, cfg); err != nil && err != context.Canceled {
		slog.Error("management server stopped", "error", err)
		os.Exit(1)
	}
}
