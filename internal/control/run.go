package control

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"time"

	"espx/internal/config"
	"espx/internal/controlplane"
	"espx/internal/costsync"
	"espx/internal/database"
	db "espx/internal/domain/db"
	"espx/internal/ingestion"
	"espx/internal/ledger"
	"espx/internal/notify"
)

func Run(ctx context.Context, cfg *config.Config, opts Options) error {
	if cfg == nil {
		return nil
	}

	var serveOpts controlplane.ServeOptions
	var closeModules func()
	needsModules := opts.Auth || opts.Billing || opts.Payment || opts.Notifier
	if needsModules {
		so, cleanups, err := buildServeOptions(ctx, cfg, opts)
		if err != nil {
			return err
		}
		serveOpts = so
		closeModules = func() {
			for i := len(cleanups) - 1; i >= 0; i-- {
				cleanups[i]()
			}
		}
		defer closeModules()
	}

	var wg sync.WaitGroup
	errCh := make(chan error, 8)

	start := func(name string, fn func(context.Context) error) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := fn(ctx); err != nil && err != context.Canceled {
				slog.Error("control component stopped", "component", name, "error", err)
				select {
				case errCh <- err:
				default:
				}
			}
		}()
	}

	if opts.MarginGuard {
		start("margin-guard", func(runCtx context.Context) error {
			return serveMarginGuard(runCtx, cfg, serveOpts.Notifier)
		})
	}
	if opts.CostSync {
		start("cost-sync", func(runCtx context.Context) error { return serveCostSync(runCtx, cfg) })
	}

	if opts.Management {
		slog.Info("control: in-process module wiring enabled")
		serveOpts.RtbBidShadeSim = ingestion.RunRtbBidShadeSim
		return controlplane.ServeWithOptions(ctx, cfg, serveOpts)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		<-done
		return ctx.Err()
	case err := <-errCh:
		return err
	case <-done:
		return nil
	}
}

func serveMarginGuard(ctx context.Context, cfg *config.Config, inProcess *controlplane.NotifierClient) error {
	pool, err := database.Connect(ctx, string(cfg.DBDSN), 10, 2)
	if err != nil {
		return err
	}
	defer pool.Close()

	queries := db.New(pool)
	registry := ingestion.NewRegistry(queries)
	registry.SetPool(pool)
	if _, err := registry.Sync(ctx); err != nil {
		slog.Warn("margin guard initial registry sync failed", "error", err)
	}
	registry.StartSync(ctx, time.Duration(cfg.RegistrySyncIntervalMs)*time.Millisecond)

	var notifierClient *controlplane.NotifierClient
	var closeNotifier func()
	if inProcess != nil {
		notifierClient = inProcess
	} else {
		notifierClient, closeNotifier, err = controlplane.TryNotifierClient(ctx, cfg)
		if err != nil {
			slog.Warn("margin guard notifier client unavailable", "error", err)
		}
		if closeNotifier != nil {
			defer closeNotifier()
		}
	}

	if !cfg.ClickHouseEnabled() {
		slog.Warn("margin guard disabled: clickhouse not configured")
		<-ctx.Done()
		return ctx.Err()
	}

	chRead, err := database.ConnectCHReadonly(ctx, string(cfg.CHReadonlyDSN))
	if err != nil {
		return err
	}
	defer chRead.Close()

	chQuery := database.NewCHQuery(chRead, database.CHQueryConfigFromApp(cfg))
	var notifierAPI notify.NotifierAPI
	if notifierClient != nil {
		notifierAPI = notifierClient.API()
	}
	worker := ledger.NewWorker(pool, chQuery, cfg, registry, notifierAPI)
	worker.Start(ctx, ledger.WorkerInterval(cfg))
	<-ctx.Done()
	return ctx.Err()
}

func serveCostSync(ctx context.Context, cfg *config.Config) error {
	pool, err := database.Connect(ctx, string(cfg.DBDSN), cfg.DBTrackerMaxConns, cfg.DBMinConns)
	if err != nil {
		return err
	}
	defer pool.Close()

	key := []byte(os.Getenv("COST_SYNC_ENCRYPTION_KEY"))
	if len(key) == 0 {
		key = []byte(os.Getenv("POSTBACK_ENCRYPTION_KEY"))
	}

	workerOpts := []costsync.WorkerOption{}
	if cfg.ClickHouseEnabled() {
		chConn, err := database.ConnectClickHouse(ctx, string(cfg.CHDSN))
		if err != nil {
			return err
		}
		defer chConn.Close()
		workerOpts = append(workerOpts, costsync.WithClickHouse(costsync.NewClickHouseInserter(chConn)))
	}

	if os.Getenv("META_APP_ID") != "" && os.Getenv("META_APP_SECRET") != "" {
		workerOpts = append(workerOpts, costsync.WithOAuth(costsync.OAuthConfig{
			MetaAppID:     os.Getenv("META_APP_ID"),
			MetaAppSecret: os.Getenv("META_APP_SECRET"),
		}))
	}
	if os.Getenv("GOOGLE_OAUTH_CLIENT_ID") != "" && os.Getenv("GOOGLE_OAUTH_CLIENT_SECRET") != "" {
		workerOpts = append(workerOpts, costsync.WithOAuth(costsync.OAuthConfig{
			GoogleClientID:     os.Getenv("GOOGLE_OAUTH_CLIENT_ID"),
			GoogleClientSecret: os.Getenv("GOOGLE_OAUTH_CLIENT_SECRET"),
		}))
	}

	worker := costsync.NewWorker(pool, key, workerOpts...)
	go worker.Start(ctx)
	<-ctx.Done()
	return ctx.Err()
}
