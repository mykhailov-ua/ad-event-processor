package ledger

import (
	"context"
	"log/slog"

	"espx/internal/config"
	"espx/internal/database"
	"espx/internal/domain"
	"espx/internal/notify"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Module struct {
	Handler *Handler
	pool    *pgxpool.Pool
	svc     *Service
	cfg     *config.Config
	cancel  context.CancelFunc
}

func (m *Module) Close() {
	if m == nil {
		return
	}
	if m.cancel != nil {
		m.cancel()
	}
	if m.pool != nil {
		m.pool.Close()
	}
}

func (m *Module) ConfigureNotifier(api notify.NotifierAPI) {
	if m == nil || m.svc == nil || m.cfg == nil || api == nil {
		return
	}
	providerName, recipient := ResolveInvoiceNotifierTarget(m.cfg)
	if providerName != "" && recipient != "" {
		m.svc.SetInvoiceDeliverer(NewNotifierInvoiceDeliverer(
			api, providerName, recipient, m.cfg.Notifier.AdminBaseURL,
		), m.cfg.Notifier.AdminBaseURL)
		m.svc.SetDriftAlerter(NewNotifierDriftAlerter(api, providerName, recipient))
	}
}

func (m *Module) StartWorkers(ctx context.Context) {
	if m == nil || m.svc == nil || m.cfg == nil || !m.cfg.Billing.InvoiceWorkerEnabled {
		return
	}
	workerCtx, cancel := context.WithCancel(ctx)
	m.cancel = cancel
	go NewInvoiceWorker(m.svc).Start(workerCtx)
	slog.Info("billing invoice worker enabled", "schedule", "1st of month 00:15 UTC")
}

func OpenAPI(ctx context.Context, cfg *config.Config) (domain.BillingAPI, func(), error) {
	noop := func() {}
	if cfg == nil || string(cfg.BillingInternalToken) == "" {
		return nil, noop, nil
	}
	token := string(cfg.BillingInternalToken)
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
	pool, err := database.Connect(ctx, string(cfg.DBDSN), cfg.DBTrackerMaxConns, cfg.DBMinConns)
	if err != nil {
		return nil, err
	}
	svc := NewService(pool)
	provider := NewPaymentProvider(cfg.Billing.PaymentProvider, string(cfg.Billing.PaymentProviderKey))
	slog.Info("billing payment provider configured", "provider", provider.Name(), "configured", provider.Configured())
	return &Module{
		Handler: NewHandler(svc, cfg),
		pool:    pool,
		svc:     svc,
		cfg:     cfg,
	}, nil
}
