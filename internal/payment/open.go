package payment

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/notify"
	checkout "ad-event-processor/internal/payment/checkout"
	settlement "ad-event-processor/internal/payment/settlement"
	webhook "ad-event-processor/internal/payment/webhook"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Module struct {
	Handler          *checkout.Handler
	pool             *pgxpool.Pool
	cfg              *config.Config
	svc              *Service
	outbox           *settlement.OutboxWorker
	settlementLedger *settlement.SettlementLedgerClient
	cryptoHold       *settlement.CryptoHoldWorker
	recon            *settlement.ReconService
	cancel           context.CancelFunc
	notifierAPI      notify.NotifierAPI
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

func (m *Module) SetNotifierAPI(api notify.NotifierAPI) {
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

	var notifierClient *settlement.NotifierClient
	if m.notifierAPI != nil {
		notifierClient = settlement.NewInProcessNotifierClient(m.notifierAPI)
	} else if m.cfg.OpsAlertsEnabled() {
		var err error
		notifierClient, _, err = settlement.ResolveNotifierClient(workerCtx, m.cfg)
		if err != nil {
			slog.Error("payment module notifier client failed", "error", err)
		}
	}

	m.outbox.SetSettlementFailedAlerter(settlement.NewSettlementFailedAlerter(notifierClient, m.cfg))
	go m.startWebhookServer(workerCtx)
	go m.outbox.Start(workerCtx, 100*time.Millisecond)
	go m.cryptoHold.Start(workerCtx, 100*time.Millisecond)

	if m.recon != nil {
		reconAlerter := settlement.NewFinancialReconAlerter(notifierClient, m.cfg)
		m.recon.SetFinancialReconAlerter(reconAlerter)
		go m.recon.StartWorker(workerCtx, time.Duration(m.cfg.PaymentFinancialReconIntervalMs)*time.Millisecond)
		slog.Info("payment financial recon worker started", "interval_ms", m.cfg.PaymentFinancialReconIntervalMs)
	}
}

func (m *Module) API(token string) checkout.PaymentAPI {
	if m == nil || m.Handler == nil {
		return nil
	}
	return checkout.NewPaymentAPI(m.Handler, token)
}

func OpenAPI(ctx context.Context, cfg *config.Config) (domain.PaymentAPI, func(), error) {
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
	checkout.LogProviderMode(cfg)
	svc := NewService(pool, cfg)
	outboxWorker := settlement.NewOutboxWorker(pool, cfg)
	settlementLedger := settlement.NewSettlementLedgerClient(cfg)
	cryptoHoldWorker := settlement.NewCryptoHoldWorker(pool, cfg)
	var reconWorker *settlement.ReconService
	if cfg.PaymentFinancialReconIntervalMs > 0 {
		reconWorker = settlement.NewReconService(pool, settlementLedger, nil)
	}
	return &Module{
		Handler:          checkout.NewHandler(svc.checkout, svc.webhook, cfg),
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
	webhook.NewWebhookHandler(m.svc.webhook, m.cfg).RegisterRoutes(mux)
	checkout.RegisterLegacyUIRoutes(mux)

	httpServer := &http.Server{
		Addr:              ":" + m.cfg.PaymentWebhookPort,
		Handler:           mux,
		ReadHeaderTimeout: time.Duration(m.cfg.HTTPReadHeaderTimeoutMs) * time.Millisecond,
		ReadTimeout:       time.Duration(m.cfg.HTTPReadTimeoutMs) * time.Millisecond,
		WriteTimeout:      time.Duration(m.cfg.HTTPWriteTimeoutMs) * time.Millisecond,
		IdleTimeout:       time.Duration(m.cfg.HTTPIdleTimeoutMs) * time.Millisecond,
	}

	go func() {
		slog.Info("starting payment webhook server", "port", m.cfg.PaymentWebhookPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("payment webhook server failed", "error", err)
		}
	}()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(ctx, time.Duration(m.cfg.Lifecycle.ShutdownTimeoutMs)*time.Millisecond)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			slog.Error("payment webhook server shutdown failed", "error", err)
		}
	}()
}
