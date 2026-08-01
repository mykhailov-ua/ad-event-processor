package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"espx/internal/config"
	"espx/internal/controlplane"
	"espx/internal/database"
	db "espx/internal/domain/db"
	"espx/internal/ingestion"
	"espx/internal/ledger"
	"espx/internal/notify"
)

func main() {
	slogLogger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(slogLogger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := database.Connect(ctx, string(cfg.DBDSN), 10, 2)
	if err != nil {
		slog.Error("failed to connect to postgres", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	queries := db.New(pool)
	registry := ingestion.NewRegistry(queries)
	registry.SetPool(pool)
	if _, err := registry.Sync(ctx); err != nil {
		slog.Warn("initial campaign registry sync failed", "error", err)
	}
	registry.StartSync(ctx, time.Duration(cfg.RegistrySyncIntervalMs)*time.Millisecond)

	notifierClient, closeNotifier, err := controlplane.TryNotifierClient(ctx, cfg)
	if err != nil {
		slog.Warn("notifier client initialization failed", "error", err)
	}
	if closeNotifier != nil {
		defer closeNotifier()
	}

	chRead, err := database.ConnectCHReadonly(ctx, string(cfg.CHReadonlyDSN))
	if err != nil {
		slog.Error("failed to connect to clickhouse readonly", "error", err)
		os.Exit(1)
	}
	defer chRead.Close()

	chQuery := database.NewCHQuery(chRead, database.CHQueryConfigFromApp(cfg))

	var notifierAPI notify.NotifierAPI
	if notifierClient != nil {
		notifierAPI = notifierClient.API()
	}
	worker := ledger.NewWorker(pool, chQuery, cfg, registry, notifierAPI)

	go worker.Start(ctx, ledger.WorkerInterval(cfg))

	slog.Info("margin guard binary started")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	slog.Info("shutting down margin guard")
}
