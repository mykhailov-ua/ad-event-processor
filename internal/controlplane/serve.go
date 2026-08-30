package controlplane

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"ad-event-processor/internal/billingadmin"
	"ad-event-processor/internal/clickhouse/migrate"
	"ad-event-processor/internal/config"
	ctrlhttp "ad-event-processor/internal/control/http"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/dedup"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/identity"
	"ad-event-processor/internal/ledger"
	"ad-event-processor/internal/licensing"
	"ad-event-processor/internal/notify"
	"ad-event-processor/internal/opsadmin"
	"ad-event-processor/internal/reconciliation"
	"ad-event-processor/internal/shardadmin"
	"ad-event-processor/pkg/netaddr"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/redis/go-redis/v9"
)

func Serve(ctx context.Context, cfg *config.Config) error {
	return ServeWithOptions(ctx, cfg, ServeOptions{})
}

func ServeWithOptions(ctx context.Context, cfg *config.Config, opts ServeOptions) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}

	postgresPools, err := database.ConnectPostgresPools(ctx, cfg)
	if err != nil {
		slog.Error("failed to connect pg pools", "error", err)
		return err
	}
	defer postgresPools.Close()
	pool := postgresPools.Read

	if cfg.MultiRegionEnabled {
		snap, snapErr := licensing.LoadDeploymentSnapshot(ctx, pool)
		if snapErr != nil || !snap.ModuleAllowed(func(f licensing.FeatureSet) bool { return f.MultiRegionEnabled() }) {
			return fmt.Errorf("multi_region requires enterprise license with multi_region feature")
		}
		slog.Info("multi-region mode enabled", "region_code", cfg.RegionCode, "cell", cfg.MultiRegionCell(), "global", cfg.MultiRegionGlobal())
	}

	var redisShards []redis.UniversalClient
	redisOpts := database.RedisShardOptions{
		PoolSize: cfg.RedisPoolSize,
	}
	redisShards, _, err = database.ConnectRedisShards(ctx, cfg, redisOpts)
	if err != nil {
		slog.Error("failed to connect to redis shards", "error", err)
		return err
	}
	licensing.StartLicenseEpochSync(ctx, shardadmin.PickHealthyControlShard(redisShards))

	sharder := domain.NewStaticSlotSharder(len(redisShards))

	controlAuthClient := opts.Auth
	tokenMaker, err := identity.NewPasetoMaker(string(cfg.TokenSymmetricKey))
	if err != nil {
		slog.Error("failed to create token maker", "error", err)
		return err
	}

	ctrlhttp.SetTrustedProxies(cfg.TrustedProxies)

	authMiddleware := NewAuthMiddleware(tokenMaker, shardadmin.PickHealthyControlShard(redisShards), cfg, controlAuthClient)
	authMiddleware.SetControlRedisShards(redisShards)
	policyStore := ctrlhttp.InitPolicyStore()
	authMiddleware.SetPolicyStore(policyStore)
	authMiddleware.SetPool(pool)
	authHandler := NewAuthHandler(controlAuthClient, tokenMaker, redisShards, cfg, authMiddleware)

	var tcpControl TCPControlPublisher
	if opts.StartControlServers != nil {
		tcpPub, closeControl, err := opts.StartControlServers(ctx, cfg, pool, sharder, len(redisShards))
		if err != nil {
			return err
		}
		if closeControl != nil {
			defer closeControl()
		}
		tcpControl = tcpPub
	}

	if cfg.MultiRegionEnabled {
		if err := ValidateScoringWeightsConfig(ctx, pool, cfg); err != nil {
			slog.Error("invalid scoring weights config", "error", err)
			return err
		}
	}

	svc := NewService(ctx, pool, redisShards, sharder, cfg)
	svc.StartBackgroundWorker(func() {
		NewShard0CatchupWorker(svc, redisOpts).Start(ctx)
	})
	slog.Info("started shard 0 catch-up worker")
	if opts.RtbBidShadeSim != nil {
		svc.SetRtbBidShadeSimulator(opts.RtbBidShadeSim)
	}
	svc.SetSettlePool(postgresPools.Settle)
	if cfg.PostgresFailoverEnabled {
		postgresFailoverRT := svc.StartPostgresFailover(ctx)
		if postgresFailoverRT != nil {
			defer postgresFailoverRT.ClosePostgresFailover()
		}
	}
	if svc == nil {
		return fmt.Errorf("management service init failed")
	}
	svc.SetPaymentPool(pool)
	if tcpControl != nil {
		svc.SetTCPControlPublisher(tcpControl)
	}

	if cfg.IsClickHouseEnabled() {
		var clickhouseWriteConn driver.Conn
		if string(cfg.ClickHouseDSN) != "" {
			var err error
			clickhouseWriteConn, err = database.ConnectClickHouse(ctx, string(cfg.ClickHouseDSN))
			if err != nil {
				slog.Error("failed to connect to clickhouse for migrations", "error", err)
				return err
			}
			if err := migrate.ApplyClickHouseMigrations(ctx, clickhouseWriteConn); err != nil {
				slog.Error("failed to apply clickhouse migrations", "error", err)
				return err
			}
			svc.SetClickHouseWrite(clickhouseWriteConn)
		}

		clickhouseReadConn, err := database.ConnectClickHouseReadonly(ctx, string(cfg.ClickHouseReadonlyDSN))
		if err != nil {
			slog.Error("failed to connect to clickhouse for reporting", "error", err)
			return err
		}
		defer func() { _ = clickhouseReadConn.Close() }()
		if clickhouseWriteConn != nil {
			defer func() { _ = clickhouseWriteConn.Close() }()
		}
		svc.SetClickHouse(clickhouseReadConn, database.ClickHouseQueryConfigFromApp(cfg))
		slog.Info("clickhouse reporting enabled", "readonly_dsn", "CH_READONLY_DSN")

		if os.Getenv("USAGE_DAILY_FLUSH_ENABLED") == "1" {
			flushInterval := 24 * time.Hour
			if v := os.Getenv("USAGE_DAILY_FLUSH_INTERVAL"); v != "" {
				if d, err := time.ParseDuration(v); err == nil && d > 0 {
					flushInterval = d
				}
			}
			svc.StartBackgroundWorker(func() {
				billingadmin.NewUsageDailyFlushWorker(pool, flushInterval).Start(ctx)
			})
			slog.Info("started usage daily flush worker", "interval", flushInterval)
		}
	}

	if pubKeyRaw := config.LicenseEnv("PUBLIC_KEY"); pubKeyRaw != "" {
		pubKey, err := licensing.ParsePublicKey([]byte(pubKeyRaw))
		if err != nil {
			slog.Error("invalid AD_EVENT_PROCESSOR_LICENSE_PUBLIC_KEY", "error", err)
			return err
		}
		if err := startLicenseWatcher(ctx, pool, redisShards, pubKey, svc); err != nil {
			return err
		}
		svc.StartLicenseRevokeQueueWorker(defaultLicenseRevokePoll)
		slog.Info("started license revoke queue worker", "interval", defaultLicenseRevokePoll)
	} else if pubKey, err := licensing.ResolvePublicKey(); err == nil {
		if err := startLicenseWatcher(ctx, pool, redisShards, pubKey, svc); err != nil {
			return err
		}
		svc.StartLicenseRevokeQueueWorker(defaultLicenseRevokePoll)
		slog.Info("started license revoke queue worker", "interval", defaultLicenseRevokePoll)
	} else if config.LicenseRequiredFromEnv() {
		slog.Error("license required but public key missing", "error", err)
		return err
	} else {
		slog.Warn("license watcher disabled", "error", err)
	}

	queries := db.New(pool)
	campaignRepo := domain.NewCampaignRepoWithDB(pool, queries)
	campaignRepo.ConfigureAuditLedgerFlush(cfg.AuditLedgerFlushSampleMask)
	customerRepo := domain.NewCustomerRepoWithDB(pool, queries)
	dedupAdapter := dedup.NewAdapter(pool, cfg.RegionCode, dedup.LoadRoutingEpoch(ctx, pool))
	var syncWorkers []*domain.SyncWorker
	for i, redisClient := range redisShards {
		if redisClient == nil {
			slog.Warn("skipping budget sync worker for unavailable redis shard", "shard", i)
			continue
		}
		sw := domain.NewSyncWorker(redisClient, campaignRepo, customerRepo, time.Duration(cfg.BudgetSyncIntervalMs)*time.Millisecond, time.Duration(cfg.LedgerBatchFlushMs)*time.Millisecond, nil, 0)
		sw.SetDedupAdapter(dedupAdapter)
		sw.ConfigureBudgetContention(
			domain.BudgetLockTTLSeconds(cfg.LedgerBatchFlushMs, cfg.BudgetSyncIntervalMs),
			cfg.QuotaStrictThresholdMicro,
		)
		syncWorkers = append(syncWorkers, sw)
		svc.StartBackgroundWorker(func() {
			sw.Start(ctx)
		})
	}

	if cfg.MultiRegionGlobal() {
		globalSpend := reconciliation.NewGlobalSpendReconciler(postgresPools.Settle, redisShards, sharder, reconciliation.GlobalSpendReconcilerConfig{
			MinBatchSize:   cfg.GlobalSpendBatchMin,
			MaxConcurrency: cfg.GlobalSpendMaxConcurrency,
		})
		svc.SetGlobalSpendReconciler(globalSpend)
		flushInterval := time.Duration(cfg.GlobalSpendFlushIntervalMs) * time.Millisecond
		svc.StartBackgroundWorker(func() {
			globalSpend.StartFlushWorker(ctx, flushInterval)
		})
		slog.Info("started global spend reconciler",
			"min_batch", cfg.GlobalSpendBatchMin,
			"flush_interval", flushInterval,
			"max_concurrency", cfg.GlobalSpendMaxConcurrency,
		)
	}

	startControlWorkers(ctx, cfg, svc, pool, postgresPools, syncWorkers)

	paymentClient, closePayment, err := openPaymentClient(ctx, cfg, opts)
	if err != nil {
		slog.Error("failed to open payment client", "error", err)
		return err
	}
	if closePayment != nil {
		defer closePayment()
	}
	if paymentClient != nil {
		if opts.Payment != nil {
			slog.Info("payment in-process client enabled")
		} else {
			slog.Info("payment module client enabled")
		}
	}

	billingClient, closeBilling, err := openBillingClient(ctx, cfg, opts)
	if err != nil {
		slog.Error("failed to open billing client", "error", err)
		return err
	}
	if closeBilling != nil {
		defer closeBilling()
	}
	if billingClient != nil {
		if opts.Billing != nil {
			slog.Info("billing in-process client enabled")
		} else {
			slog.Info("billing module client enabled")
		}
	}

	notifierClient, closeNotifier, err := openNotifierClient(ctx, cfg, opts)
	if err != nil {
		slog.Error("failed to open notifier client", "error", err)
		return err
	}
	if closeNotifier != nil {
		defer closeNotifier()
	}
	if notifierClient != nil {
		if opts.Notifier != nil {
			slog.Info("notifier in-process client enabled")
		} else {
			slog.Info("notifier module client enabled")
		}
	}
	var notifierAPI notify.NotifierAPI
	if notifierClient != nil {
		notifierAPI = notifierClient.API()
	}
	if opts.Monolith() {
		svc.SetNotifier(notifierAPI)
	}
	opsAlerter := opsadmin.NewOpsAlerter(notifierAPI, cfg)
	if opsAlerter != nil {
		svc.SetOpsAlerter(opsAlerter)
		slog.Info("ops alerts enabled")
	}

	alertmanagerWebhook := opsadmin.NewAlertmanagerWebhook(notifierClient.API(), cfg)

	if cfg.SlotMigrationEnabled {
		migrationInterval := time.Duration(cfg.SlotMigrationIntervalMs) * time.Millisecond
		orchestrator := NewSlotMigrationOrchestrator(svc, migrationInterval)
		svc.StartBackgroundWorker(func() {
			orchestrator.Start(ctx)
		})
		slog.Info("started slot migration orchestrator", "interval", migrationInterval)
	}

	if cfg.ShardOrchestratorEnabled {
		interval := time.Duration(cfg.ShardOrchestratorIntervalMs) * time.Millisecond
		shardOrch := NewShardOrchestrator(svc, &shardadmin.RealShardMetricsProvider{}, interval)
		svc.StartBackgroundWorker(func() {
			shardOrch.Start(ctx)
		})
		slog.Info("started shard orchestrator", "interval", interval)
	}

	controlHandler := NewHandler(svc, cfg, authMiddleware, controlAuthClient, paymentClient, billingClient)
	if mod, ok := opts.BillingModule.(*ledger.Module); ok && mod != nil {
		controlHandler.invoiceDelivery = billingadmin.NewInvoiceDeliveryRetryer(mod.Ledger(), redisShards)
	}

	mux := http.NewServeMux()
	opsadmin.RegisterOpsRoutes(ctx, mux, opsadmin.PlatformRoutesDeps{
		Pool:         pool,
		RedisShards:  redisShards,
		Config:       cfg,
		LicenseReady: licenseIngestReady,
	})
	if alertmanagerWebhook != nil {
		alertmanagerWebhook.Register(mux)
		slog.Info("alertmanager webhook adapter enabled")
	}
	scrapeURL := os.Getenv("OPS_METRICS_SCRAPE_URL")
	if scrapeURL == "" {
		scrapeURL = "http://127.0.0.1:" + cfg.ManagementPort + "/metrics"
	}
	svc.startOpsMetricScraper(ctx, scrapeURL)
	slog.Info("ops metric scraper enabled", "url", scrapeURL)
	svc.StartFilterRejectRollupWorker(ctx, scrapeURL)
	wireReportExportHooks()
	svc.InitReportJobRunner(reportExportDirFromWire())
	svc.StartReportJobWorker(ctx)
	svc.StartMLShadowDeltaSnapshotWorker(ctx)
	svc.StartReportScheduleWorker(ctx)

	if cfg.Management.SmartAlertsEnabled {
		interval := time.Duration(cfg.Management.SmartAlertsIntervalMin) * time.Minute
		svc.StartSmartAlertsWorker(ctx, interval)
	}
	if cfg.Management.AutomationRulesEnabled {
		svc.StartAutomationWorker(ctx, cfg.Management.AutomationRulesIntervalMin)
	}
	if cfg.Management.TrafficOptimizerEnabled {
		svc.StartTrafficOptimizerWorker(ctx, cfg.Management.TrafficOptimizerIntervalMin)
	}
	if cfg.Management.DomainHealthEnabled {
		domainInterval := time.Duration(cfg.Management.DomainHealthIntervalMin) * time.Minute
		svc.StartDomainHealthWorker(ctx, domainInterval)
	}

	authHandler.RegisterRoutes(mux)
	controlHandler.RegisterRoutes(mux)

	validateMW, err := wireOpenAPIRequestValidation(ctx, cfg)
	if err != nil {
		return err
	}

	corsMdl := ctrlhttp.NewCORSMiddleware(cfg.AllowedOrigins)
	csrfMdl := ctrlhttp.NewCSRFMiddleware(string(cfg.AdminAPIKey))
	gatewayHandler := ctrlhttp.SecurityHeadersMiddleware(corsMdl(csrfMdl(validateMW(mux))))

	slog.Info("starting management gateway server", "port", cfg.ManagementPort)

	server := &http.Server{
		Addr:              ":" + cfg.ManagementPort,
		Handler:           gatewayHandler,
		ReadHeaderTimeout: time.Duration(cfg.HTTPReadHeaderTimeoutMs) * time.Millisecond,
		ReadTimeout:       time.Duration(cfg.HTTPReadTimeoutMs) * time.Millisecond,
		WriteTimeout:      time.Duration(cfg.HTTPWriteTimeoutMs) * time.Millisecond,
		IdleTimeout:       time.Duration(cfg.HTTPIdleTimeoutMs) * time.Millisecond,
	}

	var unixSrv *http.Server
	if cfg.ControlUnixSocket != "" {
		if err := netaddr.PrepareUnixSocket(cfg.ControlUnixSocket); err != nil {
			return fmt.Errorf("control unix socket: %w", err)
		}
		unixSrv = &http.Server{
			Handler:           gatewayHandler,
			ReadHeaderTimeout: server.ReadHeaderTimeout,
			ReadTimeout:       server.ReadTimeout,
			WriteTimeout:      server.WriteTimeout,
			IdleTimeout:       server.IdleTimeout,
		}
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("management server failed", "error", err)
		}
	}()

	if unixSrv != nil {
		go func() {
			ln, err := netaddr.ListenUnix(cfg.ControlUnixSocket)
			if err != nil {
				slog.Error("management unix listen failed", "path", cfg.ControlUnixSocket, "error", err)
				return
			}
			if err := unixSrv.Serve(ln); err != nil && err != http.ErrServerClosed {
				slog.Error("management unix server failed", "error", err)
			}
		}()
		slog.Info("management unix socket enabled", "path", cfg.ControlUnixSocket)
	}

	settleHandler := NewSettlementHandler(svc, cfg)
	if opts.NotifierModule != nil {
		opts.NotifierModule.StartWorkers(ctx)
		slog.Info("notifier in-process workers enabled")
	}
	if opts.BillingModule != nil {
		if opts.Notifier != nil {
			opts.BillingModule.ConfigureNotifier(opts.Notifier.API())
		}
		opts.BillingModule.StartWorkers(ctx)
		slog.Info("billing in-process workers enabled")
	}
	if opts.PaymentModule != nil {
		opts.PaymentModule.SetSettlementAPI(settleHandler.PaymentSettlement())
		if opts.Notifier != nil {
			opts.PaymentModule.SetNotifierAPI(opts.Notifier.API())
		}
		opts.PaymentModule.StartWorkers(ctx)
		slog.Info("payment in-process settlement client enabled")
	}

	<-ctx.Done()

	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, time.Duration(cfg.Lifecycle.ShutdownTimeoutMs)*time.Millisecond)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("management server shutdown failed", "error", err)
	}
	if unixSrv != nil {
		if err := unixSrv.Shutdown(shutdownCtx); err != nil {
			slog.Error("management unix server shutdown failed", "error", err)
		}
	}

	if opsAlerter != nil {
		opsAlerter.Drain()
	}

	svc.Close()

	shardadmin.CloseConnectedRedisShards(redisShards)
	slog.Info("management server shutdown complete")
	return ctx.Err()
}
