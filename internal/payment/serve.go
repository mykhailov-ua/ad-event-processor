package payment

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"time"

	"espx/internal/config"
	"espx/internal/database"
	"espx/internal/payment/pb"

	google_grpc "google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func Serve(ctx context.Context, cfg *config.Config) error {
	if cfg == nil {
		return nil
	}

	pool, err := database.Connect(ctx, string(cfg.PaymentDBDSN), cfg.DBTrackerMaxConns, cfg.DBMinConns)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := ApplyMigrations(ctx, pool); err != nil {
		return err
	}

	prov := NewProvider(cfg)
	LogProviderMode(cfg)

	svc := NewService(pool, prov, cfg)
	grpcHandler := NewHandler(svc, cfg)

	var notifierClient *NotifierClient
	if cfg.OpsAlertsEnabled() {
		notifierClient, err = NewNotifierClient(cfg)
		if err != nil {
			return err
		}
		if notifierClient != nil {
			defer notifierClient.Close()
			slog.Info("notifier gRPC client enabled for payment ops alerts", "target", cfg.Notifier.ServerHost+":"+cfg.Notifier.Port)
		}
	}

	outboxWorker := NewOutboxWorker(pool, cfg)
	outboxWorker.SetSettlementFailedAlerter(NewSettlementFailedAlerter(notifierClient, cfg))
	go outboxWorker.Start(ctx, 100*time.Millisecond)

	cryptoHoldWorker := NewCryptoHoldWorker(pool, cfg)
	go cryptoHoldWorker.Start(ctx, 100*time.Millisecond)

	var reconWorker *ReconService
	settlementLedger := NewSettlementLedgerClient(cfg)
	defer settlementLedger.Close()
	if cfg.PaymentFinancialReconIntervalMs > 0 {
		reconAlerter := NewFinancialReconAlerter(notifierClient, cfg)
		reconWorker = NewReconService(pool, settlementLedger, reconAlerter)
		go reconWorker.StartWorker(ctx, time.Duration(cfg.PaymentFinancialReconIntervalMs)*time.Millisecond)
		slog.Info("payment financial recon worker started", "interval_ms", cfg.PaymentFinancialReconIntervalMs)
	}

	httpServerMux := http.NewServeMux()
	NewWebhookHandler(svc, cfg).RegisterRoutes(httpServerMux)
	registerLegacyUIRoutes(httpServerMux)

	httpServer := &http.Server{
		Addr:              ":" + cfg.PaymentWebhookPort,
		Handler:           httpServerMux,
		ReadHeaderTimeout: time.Duration(cfg.HttpReadHeaderTimeoutMs) * time.Millisecond,
		ReadTimeout:       time.Duration(cfg.HttpReadTimeoutMs) * time.Millisecond,
		WriteTimeout:      time.Duration(cfg.HttpWriteTimeoutMs) * time.Millisecond,
		IdleTimeout:       time.Duration(cfg.HttpIdleTimeoutMs) * time.Millisecond,
	}

	go func() {
		slog.Info("starting payment HTTP sidecar server", "port", cfg.PaymentWebhookPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP sidecar server failed", "error", err)
		}
	}()

	lis, err := net.Listen("tcp", ":"+cfg.PaymentServerPort)
	if err != nil {
		return err
	}

	grpcServer := google_grpc.NewServer()
	pb.RegisterPaymentServiceServer(grpcServer, grpcHandler)
	if cfg.Env != "production" {
		reflection.Register(grpcServer)
	}

	serveErr := make(chan error, 1)
	go func() {
		slog.Info("starting payment gRPC server", "port", cfg.PaymentServerPort)
		serveErr <- grpcServer.Serve(lis)
	}()

	select {
	case <-ctx.Done():
	case err := <-serveErr:
		if err != nil {
			return err
		}
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), time.Duration(cfg.Lifecycle.ShutdownTimeoutMs)*time.Millisecond)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("HTTP sidecar shutdown failed", "error", err)
	}
	outboxWorker.Wait()
	if reconWorker != nil {
		reconWorker.Wait()
	}

	stopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(stopped)
	}()
	select {
	case <-stopped:
	case <-shutdownCtx.Done():
		grpcServer.Stop()
	}
	return ctx.Err()
}
