package payment

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"espx/internal/config"
	"espx/internal/database"
	"espx/internal/domain"
	"espx/internal/notifier"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Module struct {
	Handler          *Handler
	pool             *pgxpool.Pool
	cfg              *config.Config
	svc              *Service
	outbox           *OutboxWorker
	settlementLedger *SettlementLedgerClient
	cryptoHold       *CryptoHoldWorker
	recon            *ReconService
	cancel           context.CancelFunc
	notifierAPI      notifier.NotifierAPI
}

func (m *Module) Close() {
	if m == nil {
		return
	}
	if m.cancel != nil {
		m.cancel()
	}
	if m.outbox != nil {
		m.outbox.Wait()
	}
	if m.recon != nil {
		m.recon.Wait()
	}
	if m.settlementLedger != nil {
		_ = m.settlementLedger.Close()
	}
	if m.pool != nil {
		m.pool.Close()
	}
}

func (m *Module) SetNotifierAPI(api notifier.NotifierAPI) {
	if m != nil {
		m.notifierAPI = api
	}
}

func (m *Module) SetSettlementAPI(api domain.PaymentSettlement) {
	if m == nil {
		return
	}
	if m.outbox != nil {
		m.outbox.SetSettlementAPI(api)
	}
	if m.settlementLedger != nil {
		m.settlementLedger.SetSettlementAPI(api)
	}
}

func (m *Module) StartWorkers(ctx context.Context) {
	if m == nil {
		return
	}
	workerCtx, cancel := context.WithCancel(ctx)
	m.cancel = cancel

	var notifierClient *NotifierClient
	if m.notifierAPI != nil {
		notifierClient = NewInProcessNotifierClient(m.notifierAPI)
	} else if m.cfg.OpsAlertsEnabled() {
		var err error
		notifierClient, _, err = ResolveNotifierClient(workerCtx, m.cfg)
		if err != nil {
			slog.Error("payment module notifier client failed", "error", err)
		}
	}

	m.outbox.SetSettlementFailedAlerter(NewSettlementFailedAlerter(notifierClient, m.cfg))
	go m.startWebhookServer(workerCtx)
	go m.outbox.Start(workerCtx, 100*time.Millisecond)
	go m.cryptoHold.Start(workerCtx, 100*time.Millisecond)

	if m.recon != nil {
		reconAlerter := NewFinancialReconAlerter(notifierClient, m.cfg)
		m.recon.alerter = reconAlerter
		go m.recon.StartWorker(workerCtx, time.Duration(m.cfg.PaymentFinancialReconIntervalMs)*time.Millisecond)
		slog.Info("payment financial recon worker started", "interval_ms", m.cfg.PaymentFinancialReconIntervalMs)
	}
}

func OpenAPI(ctx context.Context, cfg *config.Config) (PaymentAPI, func(), error) {
	noop := func() {}
	if cfg == nil || string(cfg.PaymentInternalToken) == "" {
		return nil, noop, nil
	}
	token := string(cfg.PaymentInternalToken)
	mod, err := OpenModule(ctx, cfg)
	if err != nil {
		return nil, noop, err
	}
	if mod == nil {
		return nil, noop, nil
	}
	return mod.API(token), mod.Close, nil
}

func OpenModule(ctx context.Context, cfg *config.Config) (*Module, error) {
	if cfg == nil {
		return nil, nil
	}
	pool, err := database.Connect(ctx, string(cfg.PaymentDBDSN), cfg.DBTrackerMaxConns, cfg.DBMinConns)
	if err != nil {
		return nil, err
	}
	if err := ApplyMigrations(ctx, pool); err != nil {
		pool.Close()
		return nil, err
	}
	prov := NewProvider(cfg)
	LogProviderMode(cfg)
	svc := NewService(pool, prov, cfg)
	outboxWorker := NewOutboxWorker(pool, cfg)
	settlementLedger := NewSettlementLedgerClient(cfg)
	cryptoHoldWorker := NewCryptoHoldWorker(pool, cfg)
	var reconWorker *ReconService
	if cfg.PaymentFinancialReconIntervalMs > 0 {
		reconWorker = NewReconService(pool, settlementLedger, nil)
	}
	return &Module{
		Handler:          NewHandler(svc, cfg),
		pool:             pool,
		cfg:              cfg,
		svc:              svc,
		outbox:           outboxWorker,
		settlementLedger: settlementLedger,
		cryptoHold:       cryptoHoldWorker,
		recon:            reconWorker,
	}, nil
}

func (m *Module) startWebhookServer(ctx context.Context) {
	if m == nil || m.svc == nil || m.cfg == nil {
		return
	}
	mux := http.NewServeMux()
	NewWebhookHandler(m.svc, m.cfg).RegisterRoutes(mux)
	registerLegacyUIRoutes(mux)

	httpServer := &http.Server{
		Addr:              ":" + m.cfg.PaymentWebhookPort,
		Handler:           mux,
		ReadHeaderTimeout: time.Duration(m.cfg.HttpReadHeaderTimeoutMs) * time.Millisecond,
		ReadTimeout:       time.Duration(m.cfg.HttpReadTimeoutMs) * time.Millisecond,
		WriteTimeout:      time.Duration(m.cfg.HttpWriteTimeoutMs) * time.Millisecond,
		IdleTimeout:       time.Duration(m.cfg.HttpIdleTimeoutMs) * time.Millisecond,
	}

	go func() {
		slog.Info("starting payment webhook server", "port", m.cfg.PaymentWebhookPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("payment webhook server failed", "error", err)
		}
	}()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Duration(m.cfg.Lifecycle.ShutdownTimeoutMs)*time.Millisecond)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			slog.Error("payment webhook server shutdown failed", "error", err)
		}
	}()
}
