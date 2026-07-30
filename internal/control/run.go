package control

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"time"

	"espx/internal/auth"
	"espx/internal/billing"
	"espx/internal/config"
	"espx/internal/costsync"
	"espx/internal/database"
	"espx/internal/ingestion"
	db "espx/internal/ingestion/sqlc"
	"espx/internal/management"
	"espx/internal/marginguard"
	"espx/internal/notifier"
	"espx/internal/payment"
)

func Run(ctx context.Context, cfg *config.Config, opts Options) error {
	if cfg == nil {
		return nil
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

	if opts.Auth {
		start("auth", func(runCtx context.Context) error { return auth.Serve(runCtx, cfg) })
	}
	if opts.Billing {
		start("billing", func(runCtx context.Context) error { return billing.Serve(runCtx, cfg) })
	}
	if opts.Notifier {
		start("notifier", func(runCtx context.Context) error { return notifier.Serve(runCtx, cfg) })
	}
	if opts.Payment {
		start("payment", func(runCtx context.Context) error { return payment.Serve(runCtx, cfg) })
	}
	if opts.MarginGuard {
		start("margin-guard", func(runCtx context.Context) error { return serveMarginGuard(runCtx, cfg) })
	}
	if opts.CostSync {
		start("cost-sync", func(runCtx context.Context) error { return serveCostSync(runCtx, cfg) })
	}

	if opts.Management {
		return management.Serve(ctx, cfg)
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

func serveMarginGuard(ctx context.Context, cfg *config.Config) error {
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

	notifierClient, err := management.NewNotifierClient(cfg)
	if err != nil {
		slog.Warn("margin guard notifier client unavailable", "error", err)
	}
	if notifierClient != nil {
		defer notifierClient.Close()
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
	worker := marginguard.NewWorker(pool, chQuery, cfg, registry, notifierClient)
	worker.Start(ctx, 60*time.Second)
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
		workerOpts = append(workerOpts, costsync.WithOAuthRefresher("facebook", &costsync.MetaOAuthRefresher{
			AppID:     os.Getenv("META_APP_ID"),
			AppSecret: os.Getenv("META_APP_SECRET"),
		}))
	}
	if os.Getenv("GOOGLE_OAUTH_CLIENT_ID") != "" && os.Getenv("GOOGLE_OAUTH_CLIENT_SECRET") != "" {
		workerOpts = append(workerOpts, costsync.WithOAuthRefresher("google", &costsync.GoogleOAuthRefresher{
			ClientID:     os.Getenv("GOOGLE_OAUTH_CLIENT_ID"),
			ClientSecret: os.Getenv("GOOGLE_OAUTH_CLIENT_SECRET"),
		}))
	}

	worker := costsync.NewWorker(pool, key, workerOpts...)
	go worker.Start(ctx)
	<-ctx.Done()
	return ctx.Err()
}
