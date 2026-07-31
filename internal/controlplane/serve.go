package controlplane

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"espx/internal/clickhouse/migrate"
	"espx/internal/config"
	pb "espx/internal/controlplane/pb"
	"espx/internal/database"
	"espx/internal/dedup"
	"espx/internal/domain"
	db "espx/internal/domain/db"
	"espx/internal/identity"
	"espx/internal/licensing"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

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

	var rdbs []redis.UniversalClient
	rdbs, _, err = database.ConnectRedisShards(ctx, cfg, database.RedisShardOptions{
		PoolSize: cfg.RedisPoolSize,
	})
	if err != nil {
		slog.Error("failed to connect to redis shards", "error", err)
		return err
	}

	sharder := domain.NewStaticSlotSharder(len(rdbs))

	authTarget := "127.0.0.1:" + cfg.AuthServerPort
	if host := os.Getenv("AUTH_SERVER_HOST"); host != "" {
		authTarget = host + ":" + cfg.AuthServerPort
	}

	var mgmtAuthClient *AuthClient
	var closeAuth func()
	if opts.Auth != nil {
		mgmtAuthClient = opts.Auth
		closeAuth = func() {}
	} else {
		client, closeFn, err := TryAuthClient(ctx, cfg)
		if err != nil {
			slog.Error("failed to open auth client", "error", err)
			return err
		}
		mgmtAuthClient = client
		closeAuth = closeFn
	}
	if closeAuth != nil {
		defer closeAuth()
	}
	tokenMaker, err := identity.NewPasetoMaker(string(cfg.TokenSymmetricKey))
	if err != nil {
		slog.Error("failed to create token maker", "error", err)
		return err
	}

	authMiddleware := NewAuthMiddleware(tokenMaker, PickHealthyControlShard(rdbs), cfg, mgmtAuthClient)
	authMiddleware.SetControlRedisShards(rdbs)
	policyStore := InitPolicyStore()
	authMiddleware.SetPolicyStore(policyStore)
	authMiddleware.SetPool(pool)
	authHandler := NewAuthHandler(mgmtAuthClient, tokenMaker, PickHealthyControlShard(rdbs), cfg, authMiddleware)

	if cfg.UDPControlEnabled {
		udpSrv := NewUDPControlServer(cfg, pool, sharder, len(rdbs))
		if err := udpSrv.Start(ctx); err != nil {
			slog.Error("udp control server start failed", "error", err)
			return err
		}
		defer udpSrv.Close()
	}

	var tcpSrv *TCPControlServer
	if cfg.TCPControlEnabled {
		tcpSrv = NewTCPControlServer(cfg, pool, sharder, len(rdbs))
		if err := tcpSrv.Start(ctx); err != nil {
			slog.Error("tcp control server start failed", "error", err)
			return err
		}
		defer tcpSrv.Close()
	}

	if cfg.MultiRegionEnabled {
		if err := ValidateScoringWeightsConfig(ctx, pool, cfg); err != nil {
			slog.Error("invalid scoring weights config", "error", err)
			return err
		}
	}

	svc := NewService(pool, rdbs, sharder, cfg)
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
		defer chRead.Close()
		if chWrite != nil {
			defer chWrite.Close()
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

	if pubKeyRaw := os.Getenv("ESPX_LICENSE_PUBLIC_KEY"); pubKeyRaw != "" {
		pubKey, err := licensing.ParsePublicKey([]byte(pubKeyRaw))
		if err != nil {
			slog.Error("invalid ESPX_LICENSE_PUBLIC_KEY", "error", err)
			return err
		}
		watcher := licensing.NewLicenseWatcher(pool, PickHealthyControlShard(rdbs), pubKey)
		watcher.SetControlRedisShards(rdbs)
		svc.StartBackgroundWorker(func() {
			if err := watcher.Start(ctx); err != nil && err != context.Canceled {
				slog.Error("license watcher stopped", "error", err)
			}
		})
		slog.Info("started license watcher", "mode", os.Getenv("ESPX_LICENSE_MODE"))
	}

	queries := db.New(pool)
	campaignRepo := domain.NewCampaignRepoWithDB(pool, queries)
	campaignRepo.ConfigureAuditLedgerFlush(cfg.AuditLedgerFlushSampleMask)
	customerRepo := domain.NewCustomerRepoWithDB(pool, queries)
	dedupAdapter := dedup.NewAdapter(pool, cfg.RegionCode, dedup.LoadRoutingEpoch(ctx, pool))
	var syncWorkers []*domain.SyncWorker
	for _, rdb := range rdbs {
		sw := domain.NewSyncWorker(rdb, campaignRepo, customerRepo, time.Duration(cfg.BudgetSyncIntervalMs)*time.Millisecond, time.Duration(cfg.LedgerBatchFlushMs)*time.Millisecond, nil, 0)
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
		globalSpend := NewGlobalSpendReconciler(pgPools.Settle, rdbs, sharder, GlobalSpendReconcilerConfig{
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
			brokerRedisURL = "redis://" + cfg.RedisAddrs[0] + "/0"
		}
		budgetDeltaAgg := domain.NewBudgetDeltaAggregator()
		svc.SetBrokerDeltas(budgetDeltaAgg)
		deltaConsumer := NewBudgetDeltaConsumer(budgetDeltaAgg, domain.BrokerConsumerConfig{
			BrokerAddr: cfg.Broker.URL,
			RedisURL:   brokerRedisURL,
			Topic:      cfg.BudgetDeltaTopic,
			Group:      cfg.RedisGroupName + "_mgmt_budget_delta",
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
		} else if cfg.PaymentGRPCEnabled {
			slog.Info("payment gRPC client enabled", "target", cfg.PaymentServerHost+":"+cfg.PaymentServerPort)
		} else {
			slog.Info("payment in-process module client enabled", "env", "PAYMENT_GRPC_ENABLED=0")
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
		} else if cfg.BillingGRPCEnabled {
			slog.Info("billing gRPC client enabled", "target", cfg.Billing.ServerHost+":"+cfg.Billing.Port)
		} else {
			slog.Info("billing in-process module client enabled", "env", "BILLING_GRPC_ENABLED=0")
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
		} else if cfg.NotifierGRPCEnabled {
			slog.Info("notifier gRPC client enabled", "target", cfg.Notifier.ServerHost+":"+cfg.Notifier.Port)
		} else {
			slog.Info("notifier in-process module client enabled", "env", "NOTIFIER_GRPC_ENABLED=0")
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

	mgmtHandler := NewHandler(svc, cfg, authMiddleware, mgmtAuthClient, paymentClient, billingClient)

	mux := http.NewServeMux()
	RegisterOpsRoutes(mux, pool, rdbs, cfg)
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

	authHandler.RegisterRoutes(mux)
	mgmtHandler.RegisterRoutes(mux)

	corsMdl := NewCORSMiddleware(cfg.AllowedOrigins)
	csrfMdl := NewCSRFMiddleware(string(cfg.AdminAPIKey))
	gatewayHandler := corsMdl(csrfMdl(mux))

	slog.Info("starting management gateway server", "port", cfg.ManagementPort, "auth_target", authTarget)

	server := &http.Server{
		Addr:              ":" + cfg.ManagementPort,
		Handler:           gatewayHandler,
		ReadHeaderTimeout: time.Duration(cfg.HttpReadHeaderTimeoutMs) * time.Millisecond,
		ReadTimeout:       time.Duration(cfg.HttpReadTimeoutMs) * time.Millisecond,
		WriteTimeout:      time.Duration(cfg.HttpWriteTimeoutMs) * time.Millisecond,
		IdleTimeout:       time.Duration(cfg.HttpIdleTimeoutMs) * time.Millisecond,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("management server failed", "error", err)
		}
	}()

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

	if cfg.SettlementGRPCEnabled {
		settleLis, err := net.Listen("tcp", ":"+cfg.SettlementServerPort)
		if err != nil {
			slog.Error("failed to listen on settlement port", "port", cfg.SettlementServerPort, "error", err)
			return err
		}
		settleGRPC := grpc.NewServer(grpc.UnaryInterceptor(SettlementGRPCMetricsInterceptor()))
		pb.RegisterSettlementServiceServer(settleGRPC, settleHandler)
		if cfg.Env != "production" {
			reflection.Register(settleGRPC)
		}
		go func() {
			slog.Info("starting settlement gRPC server", "port", cfg.SettlementServerPort)
			if err := settleGRPC.Serve(settleLis); err != nil {
				slog.Error("settlement gRPC server failed", "error", err)
			}
		}()

		<-ctx.Done()

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), time.Duration(cfg.Lifecycle.ShutdownTimeoutMs)*time.Millisecond)
		defer shutdownCancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			slog.Error("management server shutdown failed", "error", err)
		}

		settleStopped := make(chan struct{})
		go func() {
			settleGRPC.GracefulStop()
			close(settleStopped)
		}()
		select {
		case <-settleStopped:
			slog.Info("settlement gRPC server stopped cleanly")
		case <-shutdownCtx.Done():
			slog.Warn("settlement gRPC graceful shutdown timed out, force stopping")
			settleGRPC.Stop()
		}
	} else {
		slog.Info("settlement gRPC disabled", "env", "SETTLEMENT_GRPC_ENABLED=0")
		<-ctx.Done()

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), time.Duration(cfg.Lifecycle.ShutdownTimeoutMs)*time.Millisecond)
		defer shutdownCancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			slog.Error("management server shutdown failed", "error", err)
		}
	}

	svc.Close()

	for i, rdb := range rdbs {
		if err := rdb.Close(); err != nil {
			slog.Error("failed to close redis shard", "shard", i, "error", err)
		}
	}
	slog.Info("management server shutdown complete")
	return ctx.Err()
}
