package control

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"sync"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/controlplane"
	"ad-event-processor/internal/costsync"
	"ad-event-processor/internal/database"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/ingestion"
	"ad-event-processor/internal/ledger"
	"ad-event-processor/internal/notify"
	"ad-event-processor/internal/platformsync"
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
			if err := fn(ctx); err != nil && !errors.Is(err, context.Canceled) {
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
	if opts.PlatformCampaignSync {
		start("platform-campaign-sync", func(runCtx context.Context) error { return servePlatformCampaignSync(runCtx, cfg) })
	}

	if opts.Management {
		slog.Info("control: in-process module wiring enabled")
		serveOpts.RtbBidShadeSim = ingestion.RunRtbBidShadeSim
		err := controlplane.ServeWithOptions(ctx, cfg, serveOpts)
		wg.Wait()
		if err != nil {
			return err
		}
		return nil
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

	notifierClient := inProcess

	if !cfg.ClickHouseEnabled() {
		slog.Warn("margin guard disabled: clickhouse not configured")
		<-ctx.Done()
		return ctx.Err()
	}

	chRead, err := database.ConnectCHReadonly(ctx, string(cfg.CHReadonlyDSN))
	if err != nil {
		return err
	}
	defer func() { _ = chRead.Close() }()

	clickhouseQuery := database.NewCHQuery(chRead, database.CHQueryConfigFromApp(cfg))
	var notifierAPI notify.NotifierAPI
	if notifierClient != nil {
		notifierAPI = notifierClient.API()
	}
	worker := ledger.NewWorker(pool, clickhouseQuery, cfg, registry, notifierAPI)
	worker.Start(ctx, ledger.WorkerInterval(cfg))
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
		clickhouseConn, err := database.ConnectClickHouse(ctx, string(cfg.CHDSN))
		if err != nil {
			return err
		}
		defer func() { _ = clickhouseConn.Close() }()
		workerOpts = append(workerOpts, costsync.WithClickHouse(costsync.NewClickHouseInserter(clickhouseConn)))
		workerOpts = append(workerOpts, costsync.WithClickAttributor(costsync.NewClickCostAttributor(pool, clickhouseConn)))
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
	if os.Getenv("TIKTOK_APP_ID") != "" && os.Getenv("TIKTOK_APP_SECRET") != "" {
		workerOpts = append(workerOpts, costsync.WithOAuth(costsync.OAuthConfig{
			TikTokAppID:     os.Getenv("TIKTOK_APP_ID"),
			TikTokAppSecret: os.Getenv("TIKTOK_APP_SECRET"),
		}))
	}
	if os.Getenv("MICROSOFT_ADS_CLIENT_ID") != "" && os.Getenv("MICROSOFT_ADS_CLIENT_SECRET") != "" {
		workerOpts = append(workerOpts, costsync.WithOAuth(costsync.OAuthConfig{
			MicrosoftClientID:     os.Getenv("MICROSOFT_ADS_CLIENT_ID"),
			MicrosoftClientSecret: os.Getenv("MICROSOFT_ADS_CLIENT_SECRET"),
		}))
	}
	if os.Getenv("SNAPCHAT_CLIENT_ID") != "" && os.Getenv("SNAPCHAT_CLIENT_SECRET") != "" {
		workerOpts = append(workerOpts, costsync.WithOAuth(costsync.OAuthConfig{
			SnapchatClientID:     os.Getenv("SNAPCHAT_CLIENT_ID"),
			SnapchatClientSecret: os.Getenv("SNAPCHAT_CLIENT_SECRET"),
		}))
	}
	if os.Getenv("LINKEDIN_CLIENT_ID") != "" && os.Getenv("LINKEDIN_CLIENT_SECRET") != "" {
		workerOpts = append(workerOpts, costsync.WithOAuth(costsync.OAuthConfig{
			LinkedInClientID:     os.Getenv("LINKEDIN_CLIENT_ID"),
			LinkedInClientSecret: os.Getenv("LINKEDIN_CLIENT_SECRET"),
		}))
	}
	if os.Getenv("PINTEREST_CLIENT_ID") != "" && os.Getenv("PINTEREST_CLIENT_SECRET") != "" {
		workerOpts = append(workerOpts, costsync.WithOAuth(costsync.OAuthConfig{
			PinterestClientID:     os.Getenv("PINTEREST_CLIENT_ID"),
			PinterestClientSecret: os.Getenv("PINTEREST_CLIENT_SECRET"),
		}))
	}

	worker := costsync.NewWorker(pool, key, workerOpts...)
	worker.Start(ctx)
	return ctx.Err()
}

func servePlatformCampaignSync(ctx context.Context, cfg *config.Config) error {
	pool, err := database.Connect(ctx, string(cfg.DBDSN), cfg.DBTrackerMaxConns, cfg.DBMinConns)
	if err != nil {
		return err
	}
	defer pool.Close()

	key := []byte(os.Getenv("COST_SYNC_ENCRYPTION_KEY"))
	if len(key) == 0 {
		key = []byte(os.Getenv("POSTBACK_ENCRYPTION_KEY"))
	}

	costOpts := []costsync.WorkerOption{}
	if os.Getenv("META_APP_ID") != "" && os.Getenv("META_APP_SECRET") != "" {
		costOpts = append(costOpts, costsync.WithOAuth(costsync.OAuthConfig{
			MetaAppID:     os.Getenv("META_APP_ID"),
			MetaAppSecret: os.Getenv("META_APP_SECRET"),
		}))
	}
	if os.Getenv("GOOGLE_OAUTH_CLIENT_ID") != "" && os.Getenv("GOOGLE_OAUTH_CLIENT_SECRET") != "" {
		costOpts = append(costOpts, costsync.WithOAuth(costsync.OAuthConfig{
			GoogleClientID:     os.Getenv("GOOGLE_OAUTH_CLIENT_ID"),
			GoogleClientSecret: os.Getenv("GOOGLE_OAUTH_CLIENT_SECRET"),
		}))
	}
	costWorker := costsync.NewWorker(pool, key, costOpts...)
	platformWorker := platformsync.NewWorker(pool, key, costWorker)
	platformWorker.Start(ctx)
	return ctx.Err()
}
