// Tracker entrypoint. Package documentation: doc.go.
package main

import (
	"log/slog"
	"os"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/licensing"
	"ad-event-processor/pkg/lifecycle"
)

func main() {
	// --health-probe URL: exit 0 when /health or /ready returns 2xx (compose healthcheck).
	if len(os.Args) > 2 && os.Args[1] == "--health-probe" {
		if !lifecycle.RunHealthProbe(os.Args[2]) {
			os.Exit(1)
		}
		os.Exit(0)
	}
	if licensing.MaybeRunGuardWatchdogCLI(os.Args) {
		return
	}

	slogLogger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	slog.SetDefault(slogLogger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}
	runTracker(cfg)
}
