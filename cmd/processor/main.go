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
	"github.com/mykhailov-ua/ad-event-processor/internal/ads/repository"
	"github.com/mykhailov-ua/ad-event-processor/internal/config"
	"github.com/mykhailov-ua/ad-event-processor/internal/database"
	"github.com/mykhailov-ua/ad-event-processor/internal/infra/budget"
	infra_repo "github.com/mykhailov-ua/ad-event-processor/internal/infra/repository"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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

	// Operational DB (Postgres)
	pool, err := database.Connect(ctx, string(cfg.DBDSN), cfg.DBProcessorMaxConns, cfg.DBMinConns)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	queries := repository.New(pool)
	partManager := database.NewPartitionManager(pool, cfg.LogRetentionDays, cfg.PartitionPreCreateDays)
	partManager.StartBackground(ctx)

	// Analytics DB (ClickHouse)
	chConn, err := database.ConnectClickHouse(ctx, string(cfg.CHDSN))
	if err != nil {
		slog.Error("failed to connect to clickhouse", "error", err)
		os.Exit(1)
	}
	defer chConn.Close()

	rdb, err := database.ConnectRedis(ctx, cfg.RedisAddr, string(cfg.RedisPassword))
	if err != nil {
		slog.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}
	defer rdb.Close()

	pgStore := ads.NewPostgresStore(queries, time.Duration(cfg.WriteTimeoutMs)*time.Millisecond)
	chStore := ads.NewClickHouseStore(chConn, time.Duration(cfg.WriteTimeoutMs)*time.Millisecond)

	campaignRepo := infra_repo.NewCampaignRepo(queries)
	customerRepo := infra_repo.NewCustomerRepo(queries)

	syncWorker := budget.NewSyncWorker(rdb, campaignRepo, customerRepo, time.Duration(cfg.BudgetSyncIntervalMs)*time.Millisecond)
	go syncWorker.Start(ctx)

	pgConsumer := ads.NewStreamConsumer(
		pgStore,
		rdb,
		cfg.RedisStreamName,
		cfg.RedisGroupName+"_pg",
		cfg.RedisConsumerID,
		cfg.EventBatchSize,
		cfg.MaxWorkers,
		time.Duration(cfg.EventFlushMs)*time.Millisecond,
		time.Duration(cfg.WriteTimeoutMs)*time.Millisecond,
		time.Duration(cfg.RetryInitialWaitMs)*time.Millisecond,
		time.Duration(cfg.RetryMaxWaitMs)*time.Millisecond,
		cfg.MaxRetries,
		time.Duration(cfg.StreamMinIdleMs)*time.Millisecond,
	)
	pgConsumer.Start(ctx)

	chConsumer := ads.NewStreamConsumer(
		chStore,
		rdb,
		cfg.RedisStreamName,
		cfg.RedisGroupName+"_ch",
		cfg.RedisConsumerID,
		cfg.CHBatchSize,
		cfg.CHMaxWorkers,
		time.Duration(cfg.CHFlushIntervalMs)*time.Millisecond,
		time.Duration(cfg.WriteTimeoutMs)*time.Millisecond,
		time.Duration(cfg.RetryInitialWaitMs)*time.Millisecond,
		time.Duration(cfg.RetryMaxWaitMs)*time.Millisecond,
		cfg.MaxRetries,
		time.Duration(cfg.StreamMinIdleMs)*time.Millisecond,
	)
	chConsumer.Start(ctx)

	slog.Info("starting ad-event-processor worker", 
		"stream", cfg.RedisStreamName, 
		"pg_group", cfg.RedisGroupName+"_pg",
		"ch_group", cfg.RedisGroupName+"_ch",
		"port", cfg.ProcessorPort,
	)

	mux := http.NewServeMux()
	mux.Handle("GET /metrics", promhttp.Handler())
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	server := &http.Server{
		Addr:    ":" + cfg.ProcessorPort,
		Handler: mux,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("processor http server failed", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	slog.Info("shutting down processor")
	
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), time.Duration(cfg.ShutdownTimeoutMs)*time.Millisecond)
	defer shutdownCancel()
	
	cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("processor server shutdown failed", "error", err)
	}

	pgConsumer.Close()
	pgConsumer.Wait()
	pgStore.Close()

	chConsumer.Close()
	chConsumer.Wait()
	chStore.Close()
}
