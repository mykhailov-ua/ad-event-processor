package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/postback"
	"ad-event-processor/pkg/lifecycle"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := database.Connect(ctx, string(cfg.DBDSN), cfg.DBTrackerMaxConns, cfg.DBMinConns)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	postback.RegisterMetrics()
	metricsAddr := cfg.Postback.MetricsAddr
	if metricsAddr == "" {
		metricsAddr = "127.0.0.1:9119"
	}
	metricsSrv := lifecycle.StartMetrics(metricsAddr)
	defer func() { _ = metricsSrv.Shutdown(5 * time.Second) }()

	key := []byte(os.Getenv("POSTBACK_ENCRYPTION_KEY"))
	worker := postback.NewPostbackWorker(pool, key)
	worker.ConfigureBatchSize(cfg.PostbackBatchSize())
	worker.ConfigureStaleProcessingSec(int32(cfg.Postback.StaleProcessingSec))

	pollInterval := cfg.PostbackPollInterval()
	slog.Info("starting postback-sender daemon",
		"metrics_addr", metricsAddr,
		"poll_interval_ms", cfg.Postback.PollIntervalMs,
		"batch_size", cfg.Postback.BatchSize,
		"stale_processing_sec", cfg.Postback.StaleProcessingSec,
	)
	go worker.Start(ctx, pollInterval)

	sig := lifecycle.WaitSignal()
	slog.Info("received shutdown signal, shutting down postback-sender daemon", "signal", sig.String())
	cancel()
	slog.Info("postback-sender daemon shutdown complete")
}
