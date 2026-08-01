package notify

import (
	"context"
	"time"

	"espx/internal/config"
	"espx/internal/database"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Module struct {
	Handler *Handler
	pool    *pgxpool.Pool
	svc     *Service
	cfg     *config.Config
	bundle  ProviderBundle
	worker  *Worker
	cancel  context.CancelFunc
}

func (m *Module) Close() {
	if m == nil {
		return
	}
	if m.cancel != nil {
		m.cancel()
	}
	if m.worker != nil {
		m.worker.Wait()
	}
	if m.pool != nil {
		m.pool.Close()
	}
}

func (m *Module) StartWorkers(ctx context.Context) {
	if m == nil || m.svc == nil || m.cfg == nil {
		return
	}
	workerCtx, cancel := context.WithCancel(ctx)
	m.cancel = cancel
	go StartQueueMetricsScraper(workerCtx, m.pool, 15*time.Second)
	if m.bundle.Breakers != nil {
		go StartCircuitBreakerMetricsScraper(workerCtx, m.bundle.Breakers, 15*time.Second)
	}
	if m.cfg.Notifier.RetentionIntervalHours > 0 {
		retentionInterval := time.Duration(m.cfg.Notifier.RetentionIntervalHours) * time.Hour
		go NewRetentionJanitor(
			m.pool,
			retentionInterval,
			m.cfg.Notifier.RetentionSentDays,
			m.cfg.Notifier.RetentionFailedDays,
		).Start(workerCtx)
	}
	workerInterval := time.Duration(m.cfg.Notifier.WorkerIntervalMs) * time.Millisecond
	m.worker = NewWorker(m.svc, workerInterval, int32(m.cfg.Notifier.WorkerBatchSize))
	m.worker.StartPool(workerCtx, m.cfg.Notifier.WorkerConcurrency)
}

func OpenModule(ctx context.Context, cfg *config.Config) (*Module, error) {
	if cfg == nil {
		return nil, nil
	}
	pool, err := database.Connect(ctx, string(cfg.DBDSN), cfg.DBTrackerMaxConns, cfg.DBMinConns)
	if err != nil {
		return nil, err
	}
	if err := ApplyMigrations(ctx, pool); err != nil {
		pool.Close()
		return nil, err
	}
	RegisterMetrics()
	SetAdminBaseURL(cfg.Notifier.AdminBaseURL)
	bundle := NewProviderBundleFromConfig(cfg)
	svc := NewServiceWithOptions(pool, bundle.Providers, ServiceOptionsFromConfig(cfg))
	return &Module{
		Handler: NewHandler(svc),
		pool:    pool,
		svc:     svc,
		cfg:     cfg,
		bundle:  bundle,
	}, nil
}

func OpenAPI(ctx context.Context, cfg *config.Config) (NotifierAPI, func(), error) {
	noop := func() {}
	if cfg == nil {
		return nil, noop, nil
	}
	mod, err := OpenModule(ctx, cfg)
	if err != nil {
		return nil, noop, err
	}
	if mod == nil {
		return nil, noop, nil
	}
	return mod.API(), mod.Close, nil
}
