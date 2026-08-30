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
	// CLI probes and license watchdog exit before config.Load: no PG, Redis, or gnet on those paths.
	// --health-probe URL: exit 0 when /health or /ready returns 2xx (compose healthcheck).
	if len(os.Args) > 2 && os.Args[1] == "--health-probe" {
		if !lifecycle.RunHealthProbe(os.Args[2]) {
			os.Exit(1)
		}
		os.Exit(0)
	}
	// License guard watchdog is a separate short-lived process; tracker hot path never forks it here.
	if licensing.MaybeRunGuardWatchdogCLI(os.Args) {
		return
	}

	// Default slog LevelWarn: per-request logs stay in gnet handler; stdout is boot, config, and fatal errors.
	slogLogger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	slog.SetDefault(slogLogger)

	// config.Load reads env knobs (SERVER_PORT, TRACKER_UNIX_SOCKET, CH_INGEST_SOURCE, MAX_WORKERS, ...).
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}
	// runTracker blocks until SIGINT/SIGTERM; signal handling and drain live in wire.go, not here.
	runTracker(cfg)
}
