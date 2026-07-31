package notifier

import (
	"context"
	"log/slog"
	"net"
	"time"

	"espx/internal/config"
	"espx/internal/database"
	"espx/internal/notifier/pb"
	"espx/pkg/lifecycle"

	google_grpc "google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func Serve(ctx context.Context, cfg *config.Config) error {
	if cfg == nil {
		return nil
	}

	pool, err := database.Connect(ctx, string(cfg.DBDSN), cfg.DBTrackerMaxConns, cfg.DBMinConns)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := ApplyMigrations(ctx, pool); err != nil {
		return err
	}

	RegisterMetrics()
	SetAdminBaseURL(cfg.Notifier.AdminBaseURL)
	bundle := NewProviderBundleFromConfig(cfg)
	svc := NewServiceWithOptions(pool, bundle.Providers, ServiceOptionsFromConfig(cfg))
	grpcHandler := NewHandler(svc)

	go StartQueueMetricsScraper(ctx, pool, 15*time.Second)
	go StartCircuitBreakerMetricsScraper(ctx, bundle.Breakers, 15*time.Second)

	retentionInterval := time.Duration(cfg.Notifier.RetentionIntervalHours) * time.Hour
	go NewRetentionJanitor(
		pool,
		retentionInterval,
		cfg.Notifier.RetentionSentDays,
		cfg.Notifier.RetentionFailedDays,
	).Start(ctx)

	workerInterval := time.Duration(cfg.Notifier.WorkerIntervalMs) * time.Millisecond
	worker := NewWorker(svc, workerInterval, int32(cfg.Notifier.WorkerBatchSize))
	worker.StartPool(ctx, cfg.Notifier.WorkerConcurrency)

	metricsPort := cfg.Notifier.MetricsPort
	if metricsPort == "" {
		metricsPort = "8086"
	}
	metricsSrv := lifecycle.StartMetrics(":" + metricsPort)
	timeouts := lifecycle.TimeoutsFromConfig(cfg)

	var grpcServer *google_grpc.Server
	serveErr := make(chan error, 1)
	if cfg.NotifierGRPCEnabled {
		lis, err := net.Listen("tcp", ":"+cfg.Notifier.Port)
		if err != nil {
			return err
		}

		grpcServer = google_grpc.NewServer()
		pb.RegisterNotifierServiceServer(grpcServer, grpcHandler)
		if cfg.Env != "production" {
			reflection.Register(grpcServer)
		}

		go func() {
			slog.Info("starting notifier gRPC server", "port", cfg.Notifier.Port)
			serveErr <- grpcServer.Serve(lis)
		}()
	} else {
		slog.Info("notifier gRPC disabled", "env", "NOTIFIER_GRPC_ENABLED=0")
	}

	select {
	case <-ctx.Done():
	case err := <-serveErr:
		if err != nil {
			return err
		}
	}

	if err := lifecycle.Wait(timeouts.Wait, worker.Wait); err != nil {
		slog.Warn("notifier worker drain timed out", "error", err)
	}
	if grpcServer != nil {
		lifecycle.ShutdownGRPC(grpcServer, timeouts.Shutdown)
	}
	if err := metricsSrv.Shutdown(timeouts.Wait); err != nil {
		slog.Error("notifier metrics server shutdown failed", "error", err)
	}
	return ctx.Err()
}
