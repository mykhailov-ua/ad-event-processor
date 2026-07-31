package billing

import (
	"context"
	"log/slog"
	"net"

	"espx/internal/billing/pb"
	"espx/internal/config"
	"espx/internal/database"
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

	svc := NewService(pool)
	provider := NewPaymentProvider(cfg.Billing.PaymentProvider, string(cfg.Billing.PaymentProviderKey))
	slog.Info("billing payment provider configured", "provider", provider.Name(), "configured", provider.Configured())

	notifierAPI, closeNotifier, err := NewNotifierAPI(ctx, cfg)
	if err != nil {
		return err
	}
	if closeNotifier != nil {
		defer closeNotifier()
	}
	if notifierAPI != nil {
		providerName, recipient := ResolveInvoiceNotifierTarget(cfg)
		if providerName != "" && recipient != "" {
			svc.SetInvoiceDeliverer(NewNotifierInvoiceDeliverer(
				notifierAPI, providerName, recipient, cfg.Notifier.AdminBaseURL,
			), cfg.Notifier.AdminBaseURL)
			svc.SetDriftAlerter(NewNotifierDriftAlerter(notifierAPI, providerName, recipient))
			slog.Info("billing notifier delivery enabled", "recipient", recipient)
		}
	}

	workerCtx, workerCancel := context.WithCancel(ctx)
	if cfg.Billing.InvoiceWorkerEnabled {
		worker := NewInvoiceWorker(svc)
		go worker.Start(workerCtx)
		slog.Info("billing invoice worker enabled", "schedule", "1st of month 00:15 UTC")
	}

	grpcHandler := NewHandler(svc, cfg)

	timeouts := lifecycle.TimeoutsFromConfig(cfg)
	metricsSrv := lifecycle.StartMetrics(":" + cfg.Billing.MetricsPort)

	var grpcServer *google_grpc.Server
	serveErr := make(chan error, 1)
	if cfg.BillingGRPCEnabled {
		lis, err := net.Listen("tcp", ":"+cfg.Billing.Port)
		if err != nil {
			workerCancel()
			return err
		}

		grpcServer = google_grpc.NewServer()
		pb.RegisterBillingServiceServer(grpcServer, grpcHandler)
		if cfg.Env != "production" {
			reflection.Register(grpcServer)
		}

		go func() {
			slog.Info("starting billing gRPC server", "port", cfg.Billing.Port)
			serveErr <- grpcServer.Serve(lis)
		}()
	} else {
		slog.Info("billing gRPC disabled", "env", "BILLING_GRPC_ENABLED=0")
	}

	select {
	case <-ctx.Done():
	case err := <-serveErr:
		if err != nil {
			workerCancel()
			return err
		}
	}

	workerCancel()
	if grpcServer != nil {
		lifecycle.ShutdownGRPC(grpcServer, timeouts.Shutdown)
	}
	if err := metricsSrv.Shutdown(timeouts.Shutdown); err != nil {
		slog.Error("billing metrics server shutdown failed", "error", err)
	}
	return ctx.Err()
}
