package controlplane

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"ad-event-processor/internal/clickhouse/migrate"
	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/dedup"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/identity"
	"ad-event-processor/internal/ledger"
	"ad-event-processor/internal/licensing"
	"ad-event-processor/internal/notify"
	"ad-event-processor/pkg/httpresponse"
	"ad-event-processor/pkg/netaddr"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/redis/go-redis/v9"
)

type InProcessPaymentModule interface {
	SetSettlementAPI(api domain.PaymentSettlement)
	SetNotifierAPI(api notify.NotifierAPI)
	StartWorkers(ctx context.Context)
}

type InProcessBillingModule interface {
	ConfigureNotifier(api notify.NotifierAPI)
	StartWorkers(ctx context.Context)
}

type InProcessNotifierModule interface {
	StartWorkers(ctx context.Context)
}

type ServeOptions struct {
	Auth           *AuthClient
	Billing        *BillingClient
	Payment        *PaymentClient
	Notifier       *NotifierClient
	BillingModule  InProcessBillingModule
	PaymentModule  InProcessPaymentModule
	NotifierModule InProcessNotifierModule
	RtbBidShadeSim RtbBidShadeSimulator
}

func (o ServeOptions) Monolith() bool {
	return o.Auth != nil && o.Billing != nil && o.Payment != nil && o.Notifier != nil
}

func Serve(ctx context.Context, cfg *config.Config) error {
	return ServeWithOptions(ctx, cfg, ServeOptions{})
}

func ServeWithOptions(ctx context.Context, cfg *config.Config, opts ServeOptions) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}

	pgPools, err := database.ConnectPgPools(ctx, cfg)
	if err != nil {
		slog.Error("failed to connect pg pools", "error", err)
		return err
	}
	defer pgPools.Close()
	pool := pgPools.Read

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

	sharder := domain.NewStaticSlotSharder(len(redisShards))

	controlAuthClient := opts.Auth
	tokenMaker, err := identity.NewPasetoMaker(string(cfg.TokenSymmetricKey))
	if err != nil {
		slog.Error("failed to create token maker", "error", err)
		return err
	}

	SetTrustedProxies(cfg.TrustedProxies)

	authMiddleware := NewAuthMiddleware(tokenMaker, PickHealthyControlShard(redisShards), cfg, controlAuthClient)
	authMiddleware.SetControlRedisShards(redisShards)
	policyStore := InitPolicyStore()
	authMiddleware.SetPolicyStore(policyStore)
	authMiddleware.SetPool(pool)
	authHandler := NewAuthHandler(controlAuthClient, tokenMaker, redisShards, cfg, authMiddleware)

	if cfg.UDPControlEnabled {
		udpSrv := NewUDPControlServer(cfg, pool, sharder, len(redisShards))
		if err := udpSrv.Start(ctx); err != nil {
			slog.Error("udp control server start failed", "error", err)
			return err
		}
		defer func() { _ = udpSrv.Close() }()
	}

	var tcpSrv *TCPControlServer
	if cfg.TCPControlEnabled {
		tcpSrv = NewTCPControlServer(cfg, pool, sharder, len(redisShards))
		if err := tcpSrv.Start(ctx); err != nil {
			slog.Error("tcp control server start failed", "error", err)
			return err
		}
		defer func() { _ = tcpSrv.Close() }()
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
	svc.SetSettlePool(pgPools.Settle)
	if cfg.PgFailoverEnabled {
		pgFailoverRT := svc.StartPgFailover(ctx)
		if pgFailoverRT != nil {
			defer pgFailoverRT.ClosePgFailover()
		}
	}
	if svc == nil {
		return fmt.Errorf("management service init failed")
	}
	svc.SetPaymentPool(pool)
	if tcpSrv != nil {
		svc.SetTCPControlServer(tcpSrv)
	}

	if cfg.ClickHouseEnabled() {
		var chWrite driver.Conn
		if string(cfg.CHDSN) != "" {
			var err error
			chWrite, err = database.ConnectClickHouse(ctx, string(cfg.CHDSN))
			if err != nil {
				slog.Error("failed to connect to clickhouse for migrations", "error", err)
				return err
			}
			if err := migrate.ApplyClickHouseMigrations(ctx, chWrite); err != nil {
				slog.Error("failed to apply clickhouse migrations", "error", err)
				return err
			}
			svc.SetClickHouseWrite(chWrite)
		}

		chRead, err := database.ConnectCHReadonly(ctx, string(cfg.CHReadonlyDSN))
		if err != nil {
			slog.Error("failed to connect to clickhouse for reporting", "error", err)
			return err
		}
		defer func() { _ = chRead.Close() }()
		if chWrite != nil {
			defer func() { _ = chWrite.Close() }()
		}
		svc.SetClickHouse(chRead, database.CHQueryConfigFromApp(cfg))
		slog.Info("clickhouse reporting enabled", "readonly_dsn", "CH_READONLY_DSN")

		if os.Getenv("USAGE_DAILY_FLUSH_ENABLED") == "1" {
			flushInterval := 24 * time.Hour
			if v := os.Getenv("USAGE_DAILY_FLUSH_INTERVAL"); v != "" {
				if d, err := time.ParseDuration(v); err == nil && d > 0 {
					flushInterval = d
				}
			}
			svc.StartBackgroundWorker(func() {
				NewUsageDailyFlushWorker(pool, flushInterval).Start(ctx)
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
		globalSpend := NewGlobalSpendReconciler(pgPools.Settle, redisShards, sharder, GlobalSpendReconcilerConfig{
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

	reconInterval := time.Duration(cfg.Management.ReconIntervalMs) * time.Millisecond
	svc.StartReconWorker(reconInterval)
	slog.Info("started recon worker", "interval", reconInterval)

	volumeInterval := time.Hour
	if v := os.Getenv("VOLUME_METER_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			volumeInterval = d
		}
	}
	if os.Getenv("VOLUME_METER_ENABLED") != "0" {
		meterSource := cfg.VolumeMeterSource
		var chQ *database.CHQuery
		if meterSource == "ch" {
			chQ = svc.CHQuery()
		}
		svc.StartBackgroundWorker(func() {
			NewVolumeMeterWorker(pgPools.Settle, chQ, meterSource, volumeInterval, svc.PgGate()).Start(ctx)
		})
		slog.Info("started volume meter worker", "interval", volumeInterval, "source", meterSource)
	}

	svc.StartBackgroundWorker(func() {
		NewLedgerInvariantWorker(pgPools.Settle, cfg, nil).Start(ctx)
	})
	slog.Info("started ledger invariant worker", "interval_hours", cfg.LedgerInvariantIntervalHours)

	if cfg.BrokerEnabled() && (cfg.LocalQuotaMode == "shadow" || cfg.LocalQuotaMode == "live") {
		brokerRedisURL := cfg.Broker.RedisURL
		if brokerRedisURL == "" && len(cfg.RedisAddrs) > 0 {
			brokerRedisURL = database.BrokerRedisURL(cfg.RedisAddrs, string(cfg.RedisPassword))
		}
		budgetDeltaAgg := domain.NewBudgetDeltaAggregator()
		svc.SetBrokerDeltas(budgetDeltaAgg)
		deltaConsumer := NewBudgetDeltaConsumer(budgetDeltaAgg, domain.BrokerConsumerConfig{
			BrokerAddr: cfg.Broker.URL,
			RedisURL:   brokerRedisURL,
			Topic:      cfg.BudgetDeltaTopic,
			Group:      cfg.RedisGroupName + "_control_budget_delta",
			MaxBytes:   uint32(cfg.Broker.MaxBytes),
			Timeout:    time.Duration(cfg.Broker.TimeoutMs) * time.Millisecond,
		})
		svc.StartBackgroundWorker(func() {
			deltaConsumer.Start(ctx)
		})
		slog.Info("management budget delta recon consumer enabled", "topic", cfg.BudgetDeltaTopic)
	}

	if cfg.QuotaMode == "shadow" || cfg.QuotaMode == "live" {
		svc.StartBackgroundWorker(func() {
			NewQuotaManager(svc).Start(ctx)
		})
		slog.Info("started quota manager", "mode", cfg.QuotaMode, "chunk_size", cfg.QuotaChunkSize, "refill_threshold_pct", cfg.QuotaRefillThresholdPct)
	}

	if cfg.DeliveryOptimizerIntervalMs > 0 {
		optimizerInterval := time.Duration(cfg.DeliveryOptimizerIntervalMs) * time.Millisecond
		svc.StartDeliveryOptimizerWorker(syncWorkers, optimizerInterval)
		slog.Info("started delivery optimizer worker", "interval", optimizerInterval, "mab_interval_ms", cfg.MABIntervalMs)
	}
	if cfg.BidFloorOptimizerIntervalHours > 0 {
		floorInterval := time.Duration(cfg.BidFloorOptimizerIntervalHours) * time.Hour
		svc.StartFloorOptimizerWorker(floorInterval)
		slog.Info("started floor optimizer worker", "interval", floorInterval)
	}

	pacingInterval := time.Duration(cfg.Management.PacingIntervalMs) * time.Millisecond
	svc.StartPacingController(syncWorkers, pacingInterval)
	slog.Info("started pacing controller", "interval", pacingInterval)

	if cfg.AutoscaleIntervalMs > 0 {
		autoscaleInterval := time.Duration(cfg.AutoscaleIntervalMs) * time.Millisecond
		svc.StartAutoscaleBudgetWorker(syncWorkers, autoscaleInterval)
		slog.Info("started autoscale budget worker", "interval", autoscaleInterval)
	}

	svc.StartAuditCleaner(Days(cfg.Management.RetentionDays))
	slog.Info("started audit cleaner", "retention_days", cfg.Management.RetentionDays)

	svc.StartBackgroundWorker(func() {
		NewConsentRetentionWorker(svc).Start(ctx)
	})
	slog.Info("started consent retention worker", "retention_months", cfg.ConsentRetentionMonths)

	if cfg.EventsRetentionDays > 0 {
		svc.StartBackgroundWorker(func() {
			NewEventsRetentionWorker(pool, cfg.EventsRetentionDays).Start(ctx)
		})
		slog.Info("started events retention worker", "retention_days", cfg.EventsRetentionDays)
	}

	if cfg.ErasureWorkerIntervalMs > 0 {
		erasureInterval := time.Duration(cfg.ErasureWorkerIntervalMs) * time.Millisecond
		svc.StartBackgroundWorker(func() {
			NewErasureWorker(svc).Start(ctx, erasureInterval)
		})
		slog.Info("started privacy erasure worker", "interval", erasureInterval)
	}

	if cfg.Management.BlacklistJanitorEnabled {
		janitorInterval := time.Duration(cfg.Management.BlacklistJanitorIntervalSec) * time.Second
		svc.StartBlacklistJanitor(janitorInterval)
		slog.Info("started blacklist TTL janitor", "interval", janitorInterval)
	}

	svc.StartVendorTelemetryWorker()
	svc.StartProductTelemetryPulse()
	if cfg.TelemetryOptIn {
		slog.Info("product telemetry pulse enabled",
			"interval_sec", cfg.TelemetryIntervalSec,
			"url_configured", string(cfg.TelemetryURL) != "",
		)
	}
	if cfg.VendorTelemetryEnabled {
		slog.Info("vendor telemetry probes enabled",
			"interval_sec", cfg.VendorTelemetryIntervalSec,
			"timeout_sec", cfg.VendorTelemetryTimeoutSec,
		)
	}

	if exportPath := os.Getenv("NGINX_DENY_EXPORT_PATH"); exportPath != "" {
		nginxWorker := NewNginxConfigWorker(svc, exportPath)
		svc.StartBackgroundWorker(func() {
			nginxWorker.Start(ctx, time.Minute)
		})
		slog.Info("started nginx deny export worker", "path", exportPath)
	}

	if cfg.Management.AuditExportPath != "" {
		auditWorker := NewAuditExportWorker(svc, cfg.Management.AuditExportPath, cfg.Management.AuditExportRetentionDays)
		svc.StartBackgroundWorker(func() {
			auditWorker.Start(ctx, 24*time.Hour)
		})
		slog.Info("started audit export worker", "path", cfg.Management.AuditExportPath, "retention_days", cfg.Management.AuditExportRetentionDays)
	}

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
	opsAlerter := NewOpsAlerter(notifierClient, cfg)
	if opsAlerter != nil {
		svc.SetOpsAlerter(opsAlerter)
		slog.Info("ops alerts enabled")
	}

	alertmanagerWebhook := NewAlertmanagerWebhook(notifierClient, cfg)

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
		shardOrch := NewShardOrchestrator(svc, &RealShardMetricsProvider{}, interval)
		svc.StartBackgroundWorker(func() {
			shardOrch.Start(ctx)
		})
		slog.Info("started shard orchestrator", "interval", interval)
	}

	controlHandler := NewHandler(svc, cfg, authMiddleware, controlAuthClient, paymentClient, billingClient)
	if mod, ok := opts.BillingModule.(*ledger.Module); ok && mod != nil {
		controlHandler.invoiceDelivery = newInvoiceDeliveryRetryer(mod.Ledger(), redisShards)
	}

	mux := http.NewServeMux()
	RegisterOpsRoutes(ctx, mux, pool, redisShards, cfg)
	if alertmanagerWebhook != nil {
		alertmanagerWebhook.Register(mux)
		slog.Info("alertmanager webhook adapter enabled")
	}
	scrapeURL := os.Getenv("OPS_METRICS_SCRAPE_URL")
	if scrapeURL == "" {
		scrapeURL = "http://127.0.0.1:" + cfg.ManagementPort + "/metrics"
	}
	svc.StartOpsMetricScraper(ctx, scrapeURL)
	slog.Info("ops metric scraper enabled", "url", scrapeURL)
	svc.StartFilterRejectRollupWorker(ctx, scrapeURL)
	svc.InitReportJobRunner(reportExportDirFromWire())
	svc.StartReportJobWorker(ctx)
	svc.StartReportScheduleWorker(ctx)

	if cfg.Management.SmartAlertsEnabled {
		interval := time.Duration(cfg.Management.SmartAlertsIntervalMin) * time.Minute
		svc.StartSmartAlertsWorker(ctx, interval)
	}
	if cfg.Management.DomainHealthEnabled {
		domainInterval := time.Duration(cfg.Management.DomainHealthIntervalMin) * time.Minute
		svc.StartDomainHealthWorker(ctx, domainInterval)
	}

	authHandler.RegisterRoutes(mux)
	controlHandler.RegisterRoutes(mux)

	corsMdl := NewCORSMiddleware(cfg.AllowedOrigins)
	csrfMdl := NewCSRFMiddleware(string(cfg.AdminAPIKey))
	gatewayHandler := SecurityHeadersMiddleware(corsMdl(csrfMdl(mux)))

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

	closeConnectedRedisShards(redisShards)
	slog.Info("management server shutdown complete")
	return ctx.Err()
}

func registerAdminGoneRoutes(mux *http.ServeMux) {
	gone := func(w http.ResponseWriter, r *http.Request) {
		httpresponse.Error(w, http.StatusGone, "GONE",
			"legacy /admin HTMX routes removed; use /api/v1 JSON API (see docs/DEVELOPMENT.md)")
	}
	for _, method := range []string{"GET", "POST", "PUT", "DELETE", "PATCH"} {
		mux.HandleFunc(method+" /admin/{path...}", gone)
	}
}

func registerRootRoute(mux *http.ServeMux, gate *AdminUIGate) {
	RegisterAdminStaticRoutes(mux, gate)
}
