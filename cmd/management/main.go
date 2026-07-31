package main

import (
	"log/slog"
	"os"

	"espx/internal/config"
	"espx/internal/control"
	"espx/pkg/runtimeautotune"
)

func main() {
	if control.ProbeHealth(os.Args) {
		return
	}

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	slog.Warn("standalone binary deprecated; use cmd/control monolith")

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}
	runtimeautotune.Apply(cfg)
	control.RunLoaded(cfg, control.OptionsManagementOnly())
}
