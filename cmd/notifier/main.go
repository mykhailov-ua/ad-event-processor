package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"espx/internal/config"
	"espx/internal/notifier"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	slog.Warn("standalone binary deprecated; use cmd/control monolith")

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := notifier.Serve(ctx, cfg); err != nil && err != context.Canceled {
		slog.Error("notifier server stopped", "error", err)
		os.Exit(1)
	}
}
