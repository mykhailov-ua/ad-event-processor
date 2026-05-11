package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mykhailov-ua/ad-event-processor/internal/ads"
	ads_delivery "github.com/mykhailov-ua/ad-event-processor/internal/ads/delivery"
	"github.com/mykhailov-ua/ad-event-processor/internal/ads/repository"
	"github.com/mykhailov-ua/ad-event-processor/internal/config"
	"github.com/mykhailov-ua/ad-event-processor/internal/database"
	"github.com/mykhailov-ua/ad-event-processor/internal/infra/budget"
	infra_repo "github.com/mykhailov-ua/ad-event-processor/internal/infra/repository"
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

	// Pool is required for Registry sync and loading campaign data on budget cache miss.
	pool, err := database.Connect(ctx, string(cfg.DBDSN), cfg.DBTrackerMaxConns, cfg.DBMinConns)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	queries := repository.New(pool)
	registry := ads.NewRegistry(queries)
	count, err := registry.Sync(ctx)
	if err != nil {
		slog.Warn("initial campaign registry sync failed", "error", err)
	} else {
		slog.Info("campaign registry loaded", "campaigns", count)
	}
	registry.StartSync(ctx, time.Duration(cfg.RegistrySyncIntervalMs)*time.Millisecond)

	rdb, err := database.ConnectRedis(ctx, cfg.RedisAddr, string(cfg.RedisPassword))
	if err != nil {
		slog.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}
	defer rdb.Close()

	campaignRepo := infra_repo.NewCampaignRepo(queries)
	budgetManager := budget.NewRedisBudgetManager(rdb, campaignRepo, time.Duration(cfg.IdempotencyTTLHrs)*time.Hour)

	producer := ads.NewStreamProducer(
		rdb,
		cfg.RedisStreamName,
		cfg.StreamMaxLen,
		time.Duration(cfg.WriteTimeoutMs)*time.Millisecond,
	)

	filterEngine := ads.NewFilterEngine(
		ads.NewIPRateLimiter(rdb, cfg.RateLimitPerMin, time.Duration(cfg.RateLimitWindowMs)*time.Millisecond),
		ads.NewDuplicateEventFilter(rdb, time.Duration(cfg.DuplicateTTLSec)*time.Second),
		ads.NewBudgetFilter(budgetManager, registry, cfg.ClickAmount, cfg.ImpressionAmount),
	)

	mux := ads_delivery.NewRouter(cfg, registry, producer, filterEngine)

	slog.Info("starting ad-event-tracker", "port", cfg.ServerPort)

	server := &http.Server{
		Addr:              ":" + cfg.ServerPort,
		Handler:           mux,
		ReadHeaderTimeout: time.Duration(cfg.HttpReadHeaderTimeoutMs) * time.Millisecond,
		ReadTimeout:       time.Duration(cfg.HttpReadTimeoutMs) * time.Millisecond,
		WriteTimeout:      time.Duration(cfg.HttpWriteTimeoutMs) * time.Millisecond,
		IdleTimeout:       time.Duration(cfg.HttpIdleTimeoutMs) * time.Millisecond,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	sig := <-stop
	slog.Info("received shutdown signal", "signal", sig.String())

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), time.Duration(cfg.ShutdownTimeoutMs)*time.Millisecond)
	defer shutdownCancel()

	cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("server shutdown failed", "error", err)
	}

	registry.Wait()
}
