// Processor entrypoint. Package documentation: doc.go.
//
// Cold-path consumer wiring lives in main.go (no wire.go): Redis streams and optional mmap
// broker -> SettlementWorker (PG stats) + StreamConsumer/BrokerConsumerGroup (CH batches).
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"ad-event-processor/internal/clickhouse/migrate"
	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/dedup"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/fraud"
	ingestion "ad-event-processor/internal/ingest"
	"ad-event-processor/internal/licensing"
	"ad-event-processor/internal/metrics"
	"ad-event-processor/internal/notify"
	"ad-event-processor/internal/opsadmin"
	"ad-event-processor/internal/pgfailover"
	"ad-event-processor/internal/postback"
	"ad-event-processor/internal/rtb"
	"ad-event-processor/pkg/lifecycle"
	"ad-event-processor/pkg/logger"
	"ad-event-processor/pkg/piihash"
	rpclient "ad-event-processor/pkg/regionproxy/client"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
)

func main() {
	// CLI probes and license watchdog exit before config.Load: no PG, Redis, or consumers on those paths.
	if len(os.Args) > 2 && os.Args[1] == "--health-probe" {
		if !lifecycle.RunHealthProbe(os.Args[2]) {
			os.Exit(1)
		}
		os.Exit(0)
	}
	if licensing.MaybeRunGuardWatchdogCLI(os.Args) {
		return
	}

	// Default slog JSON info: processor is background worker; boot, settlement errors, and shutdown logs.
	slogLogger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(slogLogger)

	// config.Load: PROCESSOR_PORT, CH_INGEST_SOURCE, BROKER_*, SETTLEMENT_*, CLICKHOUSE_*.
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}
	// Retry policy shared with ingest producers (stream flush backoff).
	ingestion.SetStoreRetryPolicy(
		cfg.MaxRetries,
		time.Duration(cfg.RetryInitialWaitMs)*time.Millisecond,
		time.Duration(cfg.RetryMaxWaitMs)*time.Millisecond,
	)

	loggerCfg := logger.Config{
		LogDir:                cfg.Logger.Dir,
		FlushBufferSize:       cfg.Logger.FlushSizeKB * 1024,
		RotateSize:            int64(cfg.Logger.RotateSizeMB) * 1024 * 1024,
		RotateInterval:        cfg.Logger.RotateInterval,
		DiskLatencyLimit:      cfg.Logger.LatencyLimit,
		PersistQueueDepth:     cfg.Logger.PersistQueueDepth,
		PersistEnqueueTimeout: cfg.Logger.PersistEnqueueTimeout,
	}
	appLogger := logger.NewLogger(loggerCfg, cfg.Logger.Shards)
	defer appLogger.Close()

	logger.RegisterMetrics()
	appLogger.StartMetricsReporter(15 * time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	licensing.StartLicenseGuard(ctx, licensing.GuardConfig{
		Enabled:        licensing.GuardCompiledIn() && config.LicenseGuardEnvEnabled(),
		PtraceWatchdog: licensing.GuardCompiledIn() && config.LicenseGuardPtraceWatchdogEnabled(),
		PtraceRequired: licensing.GuardCompiledIn() && config.LicenseGuardPtraceRequired(),
	})

	if config.LicenseRequiredFromEnv() {
		licensing.StartFileLicenseRecheck(ctx, licensing.FileLicenseRecheckConfig{
			Path: config.LicensePathFromEnv(),
		})
		slog.Info("license file recheck enabled", "path", config.LicensePathFromEnv())
	}

	if err := ingestion.InitProcessorClickHouseIngestPolicy(); err != nil {
		slog.Error("failed to load processor ch ingest policy", "error", err)
		if !config.LicenseAssetsUnsealed() {
			os.Exit(1)
		}
	}

	// consumerCtx: stream/broker settlement and CH consumers; cancelled first on SIGTERM.
	consumerCtx, consumerCancel := context.WithCancel(context.Background())
	defer consumerCancel()

	// syncCtx: per-shard SyncWorker budget reconciliation; cancelled after CH/PG consumers drain.
	syncCtx, syncCancel := context.WithCancel(context.Background())
	defer syncCancel()

	// General read pool: campaign repo, dedup, partition manager, failover subscriber.
	pool, err := database.Connect(ctx, string(cfg.DBDSN), cfg.DBProcessorMaxConns, cfg.DBMinConns)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Settlement-dedicated pool; gated by ProcessorPostgresGate slot count.
	settlementPool, err := database.Connect(ctx, string(cfg.DBDSN), cfg.PostgresPoolSettleConns(cfg.SettlementLaneCount()), 1)
	if err != nil {
		slog.Error("failed to connect settlement pool", "error", err)
		os.Exit(1)
	}
	defer settlementPool.Close()

	processorPostgresGate := ingestion.NewProcessorPostgresGate(cfg.ProcessorPostgresGateSlots, cfg.PostgresPoolSettleConns(cfg.SettlementLaneCount()))
	processorClickHouseGate := ingestion.NewProcessorClickHouseGate(cfg.ProcessorClickHouseGateSlots, cfg.ClickHouseMaxConns)

	queries := db.New(pool)
	settleQueries := db.New(settlementPool)
	partManager := database.NewPartitionManager(pool, cfg.LogRetentionDays, cfg.PartitionPreCreateDays)
	partManager.StartBackground(ctx)

	if cfg.GeoIP.UpdaterEnabled {
		updater := ingestion.NewGeoIPUpdater(ingestion.GeoIPUpdaterConfig{
			DBPath:         cfg.GeoIP.DBPath,
			StagingPath:    cfg.GeoIP.StagingPath,
			EditionID:      cfg.GeoIP.EditionID,
			LicenseKey:     cfg.GeoIP.LicenseKey,
			UpdateInterval: time.Duration(cfg.GeoIP.UpdateIntervalHours) * time.Hour,
		})
		go updater.Start(ctx)
		slog.Info("geoip updater started",
			"path", cfg.GeoIP.DBPath,
			"interval_hours", cfg.GeoIP.UpdateIntervalHours,
		)
	}

	// Phase 2: optional ClickHouse conn + migrations; CH_SPOOL_DIR recovery before consumers start.
	chEnabled := cfg.IsClickHouseEnabled()
	var clickhouseConn driver.Conn
	var clickhouseStore *ingestion.ClickHouseStore
	var clickhouseJanitor *database.ClickHousePartitionJanitor
	if chEnabled {
		var err error
		clickhouseConn, err = database.ConnectClickHouse(ctx, string(cfg.ClickHouseDSN))
		if err != nil {
			slog.Error("failed to connect to clickhouse", "error", err)
			os.Exit(1)
		}
		defer func() { _ = clickhouseConn.Close() }()

		if err := migrate.ApplyClickHouseMigrations(ctx, clickhouseConn); err != nil {
			slog.Error("failed to apply clickhouse migrations", "error", err)
			os.Exit(1)
		}
	} else {
		slog.Info("clickhouse consumer disabled", "ch_consumer", "disabled")
	}

	var notifierClient *notify.Client
	if cfg.NotifierAPIEnabled() {
		api, closeNotifier, notifierErr := notify.OpenAPI(ctx, cfg)
		if notifierErr != nil {
			slog.Warn("notifier module unavailable for ops alerts", "error", notifierErr)
		} else if api != nil {
			notifierClient = notify.NewClientFromAPI(api)
			if closeNotifier != nil {
				defer closeNotifier()
			}
		}
	}
	var notifierAPI notify.NotifierAPI
	if notifierClient != nil {
		notifierAPI = notifierClient.API()
	}
	opsAlerter := opsadmin.NewOpsAlerter(notifierAPI, cfg)
	var onEmergencyDrop database.EmergencyDropAlerter
	if chEnabled && opsAlerter != nil && cfg.ClickHouseEmergencyDropPercent > 0 {
		threshold := cfg.ClickHouseEmergencyDropPercent
		onEmergencyDrop = func(table, partition string, diskPct float64) {
			opsAlerter.AlertCHEmergencyDrop(ctx, table, partition, diskPct, threshold)
		}
	}
	if chEnabled && clickhouseConn != nil && cfg.ClickHouseJanitorEnabled {
		intervalH := cfg.ClickHouseJanitorIntervalH
		if intervalH <= 0 {
			intervalH = 24
		}
		dealDays := cfg.ClickHouseRetentionDaysRtbDealOutcomes
		if dealDays <= 0 {
			dealDays = 90
		}
		exchangeDays := cfg.ClickHouseRetentionDaysRtbExchangeLog
		if exchangeDays <= 0 {
			exchangeDays = 30
		}
		clickhouseJanitor = database.NewClickHousePartitionJanitor(clickhouseConn, &database.ClickHouseJanitorOptions{
			RetentionDays: cfg.ClickHouseRawRetentionDays,
			ExtraTables: []database.ClickHouseTableRetention{
				{Table: "rtb_deal_outcomes", Days: dealDays},
				{Table: "rtb_exchange_log", Days: exchangeDays},
			},
			EmergencyDropPercent:     cfg.ClickHouseEmergencyDropPercent,
			RecompressPartsThreshold: cfg.ClickHouseRecompressPartsThreshold,
			OffPeakStartHourUTC:      cfg.ClickHouseRecompressOffPeakStartUTC,
			OffPeakEndHourUTC:        cfg.ClickHouseRecompressOffPeakEndUTC,
			OnEmergencyDrop:          onEmergencyDrop,
		})
		clickhouseJanitor.StartBackground(ctx, time.Duration(intervalH)*time.Hour)
	}

	var redisShards []redis.UniversalClient
	redisShards, _, err = database.ConnectRedisShards(ctx, cfg, database.RedisShardOptions{
		PoolSize: cfg.RedisPoolSize,
	})
	if err != nil {
		slog.Error("failed to connect to redis shards", "error", err)
		os.Exit(1)
	}

	if len(redisShards) > 0 && redisShards[0] != nil {
		licensing.StartLicenseEpochSync(ctx, redisShards[0])
	}

	streamTrimmer := ingestion.NewRedisStreamTrimmer(ingestion.RedisStreamTrimmerConfig{
		RedisShards:  redisShards,
		Streams:      []string{cfg.RedisStreamName, cfg.FraudStreamName},
		MaxLen:       cfg.StreamMaxLen,
		TrimInterval: time.Duration(cfg.RedisStreamTrimIntervalMs) * time.Millisecond,
	})
	streamTrimmer.Start(consumerCtx)

	// Phase 3: PG settlement store (gated writes); CH store wraps same settle path when clickhouse-first.
	pgStore := ingestion.NewPostgresStoreWithGate(settleQueries, time.Duration(cfg.WriteTimeoutMs)*time.Millisecond, processorPostgresGate)
	piiHasher, piiErr := piihash.NewFromSalt(cfg.PIISaltVersion, string(cfg.PIISaltHex), string(cfg.TokenSymmetricKey))
	if piiErr != nil {
		slog.Error("failed to initialize PII hasher", "error", piiErr)
		os.Exit(1)
	}
	if cfg.EventsHashIPAtInsert {
		pgStore.SetPIIHashAtInsert(piiHasher)
		slog.Info("postgres events IP hash-at-insert enabled")
	}
	var conversionPayoutApplier *ingestion.ConversionPayoutApplier
	if chEnabled && clickhouseConn != nil {
		// ClickHouseStore: batched StoreBatch via processorClickHouseGate; mmap spool on disk pressure.
		spoolCfg := ingestion.ClickHouseSpoolConfigFromConfig(cfg.ClickHouseSpoolSegmentMB, cfg.ClickHouseSpoolMaxSegments)
		spoolCfg = ingestion.ApplyClickHouseIngestPolicy(spoolCfg)
		clickhouseStore = ingestion.NewClickHouseStore(clickhouseConn, time.Duration(cfg.WriteTimeoutMs)*time.Millisecond, cfg.ClickHouseSpoolDir, spoolCfg, processorClickHouseGate)
		clickhouseStore.SetPIIHasher(piiHasher)
		conversionPayoutApplier = ingestion.NewConversionPayoutApplier(settleQueries)
		clickhouseStore.SetConversionPayoutApplier(conversionPayoutApplier)
		if err := clickhouseStore.RecoverSpool(ctx); err != nil {
			slog.Error("failed to recover clickhouse spool", "error", err)
			os.Exit(1)
		}
	}

	// clickhouse-first: PG settlement stats-only; events authoritative in CH when enabled.
	settleStore := domain.EventStore(pgStore)
	if clickhouseStore != nil {
		settleStore = ingestion.NewSettlementStore(pgStore, true)
		slog.Info("postgres settlement stats-only mode enabled (clickhouse-first)")
	}

	postbackEnqueuer := postback.NewConversionPostbackEnqueuer(settleQueries)
	pgStoreForBroker := domain.EventStore(pgStore)
	if postbackEnqueuer != nil {
		afterStored := postbackEnqueuer.OnBatchStored
		settleStore = ingestion.WrapEventStoreAfterBatch(settleStore, afterStored)
		pgStoreForBroker = ingestion.WrapEventStoreAfterBatch(pgStore, afterStored)
		slog.Info("conversion postback outbox hook enabled on PG settlement")
	}

	var fraudScorer fraud.Scorer
	if cfg.FraudMicrobatchEnabled() {
		snap, snapErr := licensing.LoadDeploymentSnapshot(ctx, pool)
		if snapErr == nil && snap.ModuleAllowed(func(f licensing.FeatureSet) bool { return f.MlFraudBoostEnabled() }) {
			var err error
			fraudScorer, err = fraud.NewLGBMScorer(cfg.FraudScoring.ModelPath)
			if err != nil {
				slog.Error("failed to initialize fraud scorer for processor micro-batching", "error", err, "path", cfg.FraudScoring.ModelPath)
				os.Exit(1)
			}
			slog.Info("initialized fraud scorer for processor micro-batching",
				"path", cfg.FraudScoring.ModelPath,
				"flush_ms", cfg.FraudScoring.MicrobatchFlushMs,
				"max_lag_sec", cfg.FraudScoring.MicrobatchMaxLagSec,
			)
		} else {
			slog.Info("ml_fraud_boost not licensed; processor fraud micro-batching disabled")
		}
	} else if cfg.FraudScoringEnabled() {
		slog.Info("FRAUD_MICROBATCH_ENABLED=false; processor fraud micro-batching disabled")
	}

	campaignRepo := ingestion.NewCampaignRepoWithDB(pool, queries)
	campaignRepo.ConfigureAuditLedgerFlush(cfg.AuditLedgerFlushSampleMask)

	var conversionDCCheck postback.ConversionDatacenterChecker
	var clickhouseQuery *database.ClickHouseQuery
	if clickhouseConn != nil {
		clickhouseQuery = database.NewClickHouseQuery(clickhouseConn, database.ClickHouseQueryConfigFromApp(cfg))
	}
	if cfg.ConversionSmartRejectEnabled() && cfg.ConversionReject.RejectDatacenterIP {
		geoProvider, geoErr := ingestion.NewMaxMindProvider(cfg.GeoIP.DBPath)
		if geoErr != nil {
			slog.Warn("conversion datacenter reject disabled: geoip country db unavailable", "error", geoErr)
		} else {
			if asnPath := cfg.GeoIP.ASNDBPath; asnPath != "" {
				if err := geoProvider.ReloadASN(asnPath); err != nil {
					slog.Warn("conversion datacenter reject: asn db load failed", "path", asnPath, "error", err)
				}
			}
			dcASNTable := ingestion.NewDCASNTable()
			if loader := ingestion.NewDCASNFeedLoader(cfg, dcASNTable); loader != nil {
				go loader.Start(ctx)
			}
			conversionDCCheck = ingestion.NewConversionDatacenterIPChecker(geoProvider, dcASNTable)
		}
	}

	var conversionRejectApplier *postback.ConversionRejectApplier
	if cfg.ConversionSmartRejectEnabled() {
		var chClickStore postback.ConversionClickStore
		if clickhouseQuery != nil {
			chClickStore = postback.NewClickHouseConversionClickStore(clickhouseQuery)
		}
		conversionRejectApplier = postback.NewConversionRejectApplier(
			cfg.ConversionReject,
			chClickStore,
			campaignRepo,
			conversionDCCheck,
		)
		if conversionRejectApplier != nil {
			settleStore = ingestion.WrapEventStoreBeforeBatch(settleStore, conversionRejectApplier.ApplyBatch)
			if clickhouseStore != nil {
				clickhouseStore.SetConversionReject(conversionRejectApplier.ApplyBatch)
			}
			slog.Info("conversion smart reject enabled on processor settlement")
		}
	}
	if conversionRejectApplier != nil && clickhouseQuery != nil && clickhouseStore != nil {
		reprocessor := postback.NewConversionRejectReprocessor(
			cfg.ConversionReject,
			clickhouseQuery,
			conversionRejectApplier,
			clickhouseStore,
			clickhouseStore,
			conversionPayoutApplier,
			postbackEnqueuer,
		)
		if reprocessor != nil {
			go reprocessor.Start(ctx)
			slog.Info("conversion smart reject reprocess worker started",
				"interval_min", cfg.ConversionReject.ReprocessIntervalMin,
				"lookback_hours", cfg.ConversionReject.ReprocessLookbackHours,
			)
		}
	}

	customerRepo := ingestion.NewCustomerRepoWithDB(pool, queries)
	dedupAdapter := dedup.NewAdapter(pool, cfg.RegionCode, dedup.LoadRoutingEpoch(ctx, pool))

	var ingestPgFailover *pgfailover.IngestRuntime
	ingestPgFailover = pgfailover.StartIngestSubscribers(ctx, redisShards, pgfailover.IngestSubscriberConfig{
		MaxConns: cfg.DBProcessorMaxConns,
		MinConns: cfg.DBMinConns,
		Interval: time.Duration(cfg.PostgresFailoverPollMs) * time.Millisecond,
	}, func(newRead *pgxpool.Pool) {
		oldRead := pool
		pool = newRead
		queries = db.New(newRead)
		campaignRepo.SetDB(newRead, queries)
		customerRepo.SetDB(newRead, queries)
		if dedupAdapter != nil {
			dedupAdapter.SetPool(newRead)
		}
		partManager.SetPool(newRead)
		if oldRead != nil && oldRead != newRead {
			oldRead.Close()
		}

		dsn := ingestPgFailover.CurrentDSN()
		if dsn == "" {
			dsn = string(cfg.DBDSN)
		}
		settleNew, connectErr := database.Connect(ctx, dsn, cfg.PostgresPoolSettleConns(cfg.SettlementLaneCount()), 1)
		if connectErr != nil {
			slog.Warn("processor pg failover settle pool reconnect failed", "error", connectErr)
			return
		}
		oldSettle := settlementPool
		settlementPool = settleNew
		settleQueries = db.New(settleNew)
		pgStore.SetQuerier(settleQueries)
		if postbackEnqueuer != nil {
			postbackEnqueuer.SetStore(settleQueries)
		}
		if conversionPayoutApplier != nil {
			conversionPayoutApplier.SetStore(settleQueries)
		}
		if oldSettle != nil && oldSettle != settleNew && oldSettle != oldRead {
			oldSettle.Close()
		}
		slog.Info("processor pg failover reconnected read and settle pools")
	})
	if ingestPgFailover != nil {
		defer ingestPgFailover.Stop()
	}

	var weightCtrl *ingestion.ProcessorWeightController
	if cfg.ProcessorWeightEnabled {
		weightCtrl = ingestion.NewProcessorWeightController(ingestion.ProcessorWeightConfigFromApp(cfg), processorPostgresGate, nil)
		weightCtrl.Start(ctx)
		slog.Info("processor weight scheduling enabled", "node_id", cfg.NodeID)
	}

	// Phase 5: per-shard background workers (started below; Close/Wait on shutdown).
	var pgSettlementWorkers []*ingestion.SettlementWorker
	var chConsumers []*ingestion.StreamConsumer
	var brokerConsumers []*ingestion.BrokerStreamConsumer
	var brokerCHGroup *BrokerConsumerGroup
	var brokerFraudGroup *BrokerConsumerGroup
	var brokerReconcile *ingestion.BrokerReconcileWorker
	var budgetDeltaConsumer *rtb.BudgetDeltaConsumer
	var syncWorkers []*ingestion.SyncWorker
	var fraudMicrobatcher *fraud.MicroBatcher
	if fraudScorer != nil && len(redisShards) > 0 {
		mbCfg := fraud.MicroBatcherConfig{
			FlushInterval:   time.Duration(cfg.FraudScoring.MicrobatchFlushMs) * time.Millisecond,
			MaxStreamLagSec: float64(cfg.FraudScoring.MicrobatchMaxLagSec),
		}
		fraudMicrobatcher = fraud.NewMicroBatcher(redisShards, fraudScorer, cfg.CampaignUpdateChannel, mbCfg)
		go fraudMicrobatcher.Start(consumerCtx)
	}

	var spendSyncProducer *ingestion.SpendSyncProducer
	if cfg.MultiRegionCell() {
		if cfg.RegionProxyAddr == "" {
			slog.Error("regional processor requires REGION_PROXY_ADDR when MULTI_REGION_ENABLED=1")
			os.Exit(1)
		}
		rpClient := rpclient.New(rpclient.Config{
			Addr:     cfg.RegionProxyAddr,
			RedisURL: cfg.RegionProxyRedisURL,
		})
		defer func() { _ = rpClient.Close() }()
		spendSyncProducer = ingestion.NewSpendSyncProducer(newRegionProxySpendSync(rpClient), cfg.GlobalSpendBatchMin)
		slog.Info("regional spend sync producer enabled",
			"region", cfg.RegionCode,
			"proxy_addr", cfg.RegionProxyAddr,
			"min_batch", cfg.GlobalSpendBatchMin,
		)
	}

	// Per-shard workers: SyncWorker (budget), SettlementWorker (PG stream), StreamConsumer (_ch/_fraud).
	// CH_INGEST_SOURCE=broker disables Redis SettlementWorker and _ch/_fraud StreamConsumers on this path.
	for i, redisClient := range redisShards {
		shardID := fmt.Sprintf("shard_%d", i)

		sw := ingestion.NewSyncWorker(redisClient, campaignRepo, customerRepo, time.Duration(cfg.BudgetSyncIntervalMs)*time.Millisecond, time.Duration(cfg.LedgerBatchFlushMs)*time.Millisecond, processorPostgresGate, 0)
		sw.SetDedupAdapter(dedupAdapter)
		sw.ConfigureBudgetContention(
			ingestion.BudgetLockTTLSeconds(cfg.LedgerBatchFlushMs, cfg.BudgetSyncIntervalMs),
			cfg.QuotaStrictThresholdMicro,
		)
		if spendSyncProducer != nil {
			sw.SetSpendSyncProducer(spendSyncProducer)
		}
		syncWorkers = append(syncWorkers, sw)
		sw.Start(syncCtx)

		if !cfg.BrokerPrimaryCH() {
			// SettlementWorker: XREADGROUP on REDIS_STREAM_NAME, consumer group REDIS_GROUP_NAME_pg.
			settleFlush := time.Duration(cfg.SettlementFlushMs) * time.Millisecond
			settleW := ingestion.NewSettlementWorker(
				settleStore,
				redisClient,
				cfg.RedisStreamName,
				cfg.RedisGroupName+"_pg",
				cfg.RedisConsumerID+"_"+shardID,
				cfg.SettlementLaneCount(),
				cfg.EventBatchSize,
				settleFlush,
				time.Duration(cfg.WriteTimeoutMs)*time.Millisecond,
				time.Duration(cfg.RetryInitialWaitMs)*time.Millisecond,
				time.Duration(cfg.RetryMaxWaitMs)*time.Millisecond,
				cfg.MaxRetries,
				time.Duration(cfg.StreamMinIdleMs)*time.Millisecond,
				time.Duration(cfg.Lifecycle.DrainTimeoutMs)*time.Millisecond,
			)
			settleW.SetLogger(appLogger)
			settleW.SetAuditLogSampleMask(cfg.AuditLogSampleMask)
			if weightCtrl != nil {
				settleW.SetWeightController(weightCtrl)
			}
			segmentHandler := ingestion.NewSegmentConversionHandler(campaignRepo, queries, []redis.UniversalClient{redisClient}, piiHasher)
			settleW.SetOnMessageProcessed(segmentHandler.Handle)
			pgSettlementWorkers = append(pgSettlementWorkers, settleW)
			settleW.Start(consumerCtx)
		} else {
			slog.Info("processor: Redis SettlementWorker disabled (CH_INGEST_SOURCE=broker)")
		}

		if clickhouseStore != nil && !cfg.BrokerPrimaryCH() {
			// StreamConsumer _ch: same ad:events stream as settlement; separate consumer group for CH batch ingest.
			cc := ingestion.NewStreamConsumer(
				clickhouseStore,
				redisClient,
				cfg.RedisStreamName,
				cfg.RedisGroupName+"_ch",
				cfg.RedisConsumerID+"_"+shardID,
				cfg.ClickHouseBatchSize,
				cfg.ProcessorClickHouseStreamWorkers(),
				time.Duration(cfg.ClickHouseFlushIntervalMs)*time.Millisecond,
				time.Duration(cfg.WriteTimeoutMs)*time.Millisecond,
				time.Duration(cfg.RetryInitialWaitMs)*time.Millisecond,
				time.Duration(cfg.RetryMaxWaitMs)*time.Millisecond,
				cfg.MaxRetries,
				time.Duration(cfg.StreamMinIdleMs)*time.Millisecond,
				time.Duration(cfg.Lifecycle.DrainTimeoutMs)*time.Millisecond,
			)
			cc.SetLogger(appLogger)
			cc.SetAuditLogSampleMask(cfg.AuditLogSampleMask)
			if weightCtrl != nil {
				cc.SetWeightController(weightCtrl)
			}

			if fraudMicrobatcher != nil {
				cc.SetOnMessageProcessed(fraudMicrobatcher.Enqueue)
			}

			chConsumers = append(chConsumers, cc)
			cc.Start(consumerCtx)
		} else if clickhouseStore != nil && cfg.BrokerPrimaryCH() {
			slog.Info("processor: Redis _ch StreamConsumer disabled (CH_INGEST_SOURCE=broker)")
		}

		if clickhouseStore != nil && !cfg.BrokerPrimaryCH() {
			// StreamConsumer _fraud: FRAUD_STREAM_NAME lane; no OnMessageProcessed hook (no microbatch here).
			fc := ingestion.NewStreamConsumer(
				clickhouseStore,
				redisClient,
				cfg.FraudStreamName,
				cfg.RedisGroupName+"_fraud",
				cfg.RedisConsumerID+"_fraud_"+shardID,
				cfg.ClickHouseBatchSize,
				cfg.ProcessorClickHouseStreamWorkers(),
				time.Duration(cfg.ClickHouseFlushIntervalMs)*time.Millisecond,
				time.Duration(cfg.WriteTimeoutMs)*time.Millisecond,
				time.Duration(cfg.RetryInitialWaitMs)*time.Millisecond,
				time.Duration(cfg.RetryMaxWaitMs)*time.Millisecond,
				cfg.MaxRetries,
				time.Duration(cfg.StreamMinIdleMs)*time.Millisecond,
				time.Duration(cfg.Lifecycle.DrainTimeoutMs)*time.Millisecond,
			)
			fc.SetLogger(appLogger)
			fc.SetAuditLogSampleMask(cfg.AuditLogSampleMask)
			if weightCtrl != nil {
				fc.SetWeightController(weightCtrl)
			}
			chConsumers = append(chConsumers, fc)
			fc.Start(consumerCtx)
		}
	}

	if cfg.BrokerPrimaryCH() {
		metrics.IngestFraudPath.Set(1)
	} else {
		metrics.IngestFraudPath.Set(0)
	}

	if !cfg.BrokerPrimaryCH() {
		ingestion.StartFraudLagPublisher(
			ctx,
			redisShards,
			cfg.FraudStreamName,
			cfg.RedisGroupName+"_fraud",
			cfg.FraudConsumerLagSec,
			2*time.Second,
		)
	}

	// Phase 6: mmap WAL broker bridge when BROKER_ENABLED=1.
	// PG partition consumers always; CH/fraud BrokerConsumerGroup when clickhouseStore set;
	// reconcile worker only when CH_INGEST_SOURCE != broker (shadow/dual-path divergence).
	if cfg.BrokerEnabled() {
		brokerRedisURL := cfg.Broker.RedisURL
		if brokerRedisURL == "" && len(cfg.RedisAddrs) > 0 {
			brokerRedisURL = database.BrokerRedisURL(cfg.RedisAddrs, string(cfg.RedisPassword))
		}
		brokerBase := ingestion.BrokerConsumerConfig{
			BrokerAddr: cfg.Broker.URL,
			RedisURL:   brokerRedisURL,
			Topic:      cfg.Broker.Topic,
			BatchSize:  cfg.EventBatchSize,
			FlushInt:   time.Duration(cfg.EventFlushMs) * time.Millisecond,
			MaxBytes:   uint32(cfg.Broker.MaxBytes),
			Timeout:    time.Duration(cfg.Broker.TimeoutMs) * time.Millisecond,
			ShadowMode: cfg.Broker.ShadowMode,
		}
		partCount := cfg.Broker.PartitionCount
		if partCount <= 0 {
			partCount = 1
		}
		writeTimeout := time.Duration(cfg.WriteTimeoutMs) * time.Millisecond
		retryInit := time.Duration(cfg.RetryInitialWaitMs) * time.Millisecond
		retryMax := time.Duration(cfg.RetryMaxWaitMs) * time.Millisecond

		for p := range partCount {
			// BrokerStreamConsumer _pg_broker: PG settlement from broker topic (parallel to Redis stream path).
			pgBrokerCfg := brokerBase
			pgBrokerCfg.Partition = uint16(p)
			pgBrokerCfg.Group = cfg.RedisGroupName + "_pg_broker"
			pgBroker := ingestion.NewBrokerStreamConsumer(pgStoreForBroker, pgBrokerCfg, writeTimeout, retryInit, retryMax, cfg.MaxRetries)
			pgBroker.SetDedupAdapter(dedupAdapter)
			pgBroker.SetLogger(appLogger)
			brokerConsumers = append(brokerConsumers, pgBroker)
			pgBroker.Start(consumerCtx)
		}

		if clickhouseStore != nil {
			// BrokerConsumerGroup _ch_broker: authoritative CH ingest when CH_INGEST_SOURCE=broker.
			chGroupCfg := BrokerConsumerGroupConfig{
				BrokerAddr:     cfg.Broker.URL,
				RedisURL:       brokerRedisURL,
				Topic:          cfg.Broker.Topic,
				Group:          cfg.RedisGroupName + "_ch_broker",
				PartitionCount: partCount,
				BatchSize:      cfg.ClickHouseBatchSize,
				FlushInterval:  time.Duration(cfg.ClickHouseFlushIntervalMs) * time.Millisecond,
				MaxBytes:       uint32(cfg.Broker.MaxBytes),
				Timeout:        time.Duration(cfg.Broker.TimeoutMs) * time.Millisecond,
				DataDir:        cfg.Logger.Dir + "/offsets",
				ShadowMode:     cfg.Broker.ShadowMode,
			}
			if fraudMicrobatcher != nil {
				chGroupCfg.OnMessageProcessed = func(evt *domain.Event, _ uint64) {
					if evt == nil {
						return
					}
					fraudMicrobatcher.Enqueue(evt, "")
				}
			}
			var chGrpErr error
			brokerCHGroup, chGrpErr = NewBrokerConsumerGroup(clickhouseStore, chGroupCfg, appLogger)
			if chGrpErr != nil {
				slog.Error("failed to create broker consumer group for clickhouse", "error", chGrpErr)
			} else {
				brokerCHGroup.Start(consumerCtx)
			}
		}

		if clickhouseStore != nil && cfg.BrokerPrimaryCH() {
			slog.Info("processor: Redis _fraud StreamConsumer disabled (CH_INGEST_SOURCE=broker)")
			// BrokerConsumerGroup _fraud_broker: fraud topic when broker is sole CH ingest path.
			fraudGroupCfg := BrokerConsumerGroupConfig{
				BrokerAddr:     cfg.Broker.URL,
				RedisURL:       brokerRedisURL,
				Topic:          cfg.Broker.FraudTopic,
				Group:          cfg.RedisGroupName + "_fraud_broker",
				PartitionCount: partCount,
				BatchSize:      cfg.ClickHouseBatchSize,
				FlushInterval:  time.Duration(cfg.ClickHouseFlushIntervalMs) * time.Millisecond,
				MaxBytes:       uint32(cfg.Broker.MaxBytes),
				Timeout:        time.Duration(cfg.Broker.TimeoutMs) * time.Millisecond,
				DataDir:        cfg.Logger.Dir + "/offsets/fraud",
				ShadowMode:     cfg.Broker.ShadowMode,
			}
			var fraudGrpErr error
			brokerFraudGroup, fraudGrpErr = NewBrokerConsumerGroup(clickhouseStore, fraudGroupCfg, appLogger)
			if fraudGrpErr != nil {
				slog.Error("failed to create broker consumer group for fraud clickhouse", "error", fraudGrpErr)
			} else {
				brokerFraudGroup.Start(consumerCtx)
			}
		}

		if len(redisShards) > 0 && !cfg.BrokerPrimaryCH() {
			brokerReconcile = ingestion.NewBrokerReconcileWorker(ingestion.BrokerReconcileConfig{
				BrokerAddr:          cfg.Broker.URL,
				BrokerRedis:         brokerRedisURL,
				Topic:               cfg.Broker.Topic,
				PartitionCount:      partCount,
				BrokerGroup:         cfg.RedisGroupName + "_pg_broker",
				StreamName:          cfg.RedisStreamName,
				Interval:            time.Duration(cfg.Broker.ReconcileIntervalMs) * time.Millisecond,
				DivergenceThreshold: cfg.Broker.DivergenceThreshold,
			}, redisShards)
			brokerReconcile.Start(consumerCtx)
		} else if cfg.BrokerPrimaryCH() {
			slog.Info("processor: broker reconcile skipped (CH_INGEST_SOURCE=broker)")
		}

		slog.Info("broker ingest bridge enabled",
			"broker", cfg.Broker.URL,
			"topic", cfg.Broker.Topic,
			"fraud_topic", cfg.Broker.FraudTopic,
			"partitions", partCount,
			"shadow_mode", cfg.Broker.ShadowMode,
			"pg_group", cfg.RedisGroupName+"_pg_broker",
			"ch_group", cfg.RedisGroupName+"_ch_broker",
			"fraud_group", cfg.RedisGroupName+"_fraud_broker",
		)
	}

	if cfg.BrokerEnabled() && (cfg.LocalQuotaMode == "shadow" || cfg.LocalQuotaMode == "live") {
		// BudgetDeltaConsumer: replays tracker local-quota broker deltas into processor budget sync.
		brokerRedisURL := cfg.Broker.RedisURL
		if brokerRedisURL == "" && len(cfg.RedisAddrs) > 0 {
			brokerRedisURL = database.BrokerRedisURL(cfg.RedisAddrs, string(cfg.RedisPassword))
		}
		budgetDeltaConsumer = rtb.NewBudgetDeltaConsumer(
			domain.NewBudgetDeltaAggregator(),
			rtb.BudgetDeltaConsumerConfig{
				BrokerAddr: cfg.Broker.URL,
				RedisURL:   brokerRedisURL,
				Topic:      cfg.BudgetDeltaTopic,
				Group:      cfg.RedisGroupName + "_budget_delta",
				MaxBytes:   uint32(cfg.Broker.MaxBytes),
				Timeout:    time.Duration(cfg.Broker.TimeoutMs) * time.Millisecond,
			},
		)
		budgetDeltaConsumer.Start(consumerCtx)
		slog.Info("budget delta consumer enabled", "topic", cfg.BudgetDeltaTopic)
	}

	slog.Info("starting ad-event-processor worker",
		"stream", cfg.RedisStreamName,
		"pg_group", cfg.RedisGroupName+"_pg",
		"ch_enabled", chEnabled,
		"port", cfg.ProcessorPort,
	)
	if !chEnabled {
		slog.Info("clickhouse stream consumers skipped", "ch_consumer", "disabled")
	}

	// Phase 7: HTTP sidecar on PROCESSOR_PORT (default 8186): /metrics, /health, /ready.
	// Readiness fails when PG/CH/Redis ping fails, CH spool over max segments, or stream lag exceeds cap.
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", promhttp.Handler())
	live := &lifecycle.Liveness{}
	ready := &lifecycle.ReadinessProbe{}
	ready.StartBackground(ctx, 2*time.Second, func(probeCtx context.Context) bool {
		if err := pool.Ping(probeCtx); err != nil {
			return false
		}
		if clickhouseConn != nil {
			if err := clickhouseConn.Ping(probeCtx); err != nil {
				return false
			}
		}
		for i, redisClient := range redisShards {
			if err := redisClient.Ping(probeCtx).Err(); err != nil {
				return false
			}
			if cfg.RedisStreamName != "" {
				if n, err := redisClient.XLen(probeCtx, cfg.RedisStreamName).Result(); err == nil {
					metrics.ProcessorStreamXLen.WithLabelValues(strconv.Itoa(i)).Set(float64(n))
				}
			}
		}
		if clickhouseStore != nil {
			if spool := clickhouseStore.Spool(); spool != nil {
				seg := spool.SegmentCount()
				metrics.ClickHouseSpoolSegments.Set(float64(seg))
				if seg > cfg.ClickHouseSpoolMaxSegments {
					return false
				}
			}
		}
		if cfg.ProcessorStreamLagMaxSec > 0 && ingestion.ProcessorStreamLagSec() > int64(cfg.ProcessorStreamLagMaxSec) {
			return false
		}
		return true
	})
	lifecycle.Register(mux, live, ready)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		ready.ServeReadyz(w, r)
	})

	server := &http.Server{
		Addr:    ":" + cfg.ProcessorPort,
		Handler: mux,
	}
	lifecycle.ApplySidecarHTTPServerTimeouts(server)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("processor http server failed", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	slog.Info("shutting down processor")

	// Shutdown drain (LIFECYCLE_SHUTDOWN_TIMEOUT_MS / LIFECYCLE_WAIT_TIMEOUT_MS):
	// 1) consumerCancel stops stream/broker fetch loops
	// 2) broker Close then HTTP server Shutdown
	// 3) broker Wait, settlement Wait, CH consumers Wait, clickhouseStore.Close
	// 4) syncCancel + SyncWorker Wait, partition manager, stream trimmer, Redis Close
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), time.Duration(cfg.Lifecycle.ShutdownTimeoutMs)*time.Millisecond)
	defer shutdownCancel()

	consumerCancel()

	for _, bc := range brokerConsumers {
		bc.Close()
	}
	if brokerCHGroup != nil {
		brokerCHGroup.Close()
	}
	if brokerFraudGroup != nil {
		brokerFraudGroup.Close()
	}
	if budgetDeltaConsumer != nil {
		budgetDeltaConsumer.Close()
	}
	if brokerReconcile != nil {
		brokerReconcile.Close()
	}

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("processor server shutdown failed", "error", err)
	}

	waitCtx, waitCancel := context.WithTimeout(context.Background(), time.Duration(cfg.Lifecycle.WaitTimeoutMs)*time.Millisecond)
	defer waitCancel()

	for _, bc := range brokerConsumers {
		if err := bc.Wait(waitCtx); err != nil {
			slog.Error("broker consumer wait failed", "error", err)
		}
	}
	if brokerCHGroup != nil {
		if err := brokerCHGroup.Wait(waitCtx); err != nil {
			slog.Error("broker clickhouse consumer group wait failed", "error", err)
		}
	}
	if brokerFraudGroup != nil {
		if err := brokerFraudGroup.Wait(waitCtx); err != nil {
			slog.Error("broker fraud clickhouse consumer group wait failed", "error", err)
		}
	}
	if brokerReconcile != nil {
		if err := brokerReconcile.Wait(waitCtx); err != nil {
			slog.Error("broker reconcile wait failed", "error", err)
		}
	}

	for _, sw := range pgSettlementWorkers {
		sw.Close()
		if err := sw.Wait(waitCtx); err != nil {
			slog.Error("pg settlement worker wait failed", "error", err)
		}
	}
	_ = pgStore.Close()

	for _, cc := range chConsumers {
		cc.Close()
		if err := cc.Wait(waitCtx); err != nil {
			slog.Error("ch consumer wait failed", "error", err)
		}
	}
	if clickhouseStore != nil {
		_ = clickhouseStore.Close()
	}

	syncCancel()
	for i, sw := range syncWorkers {
		if err := sw.Wait(waitCtx); err != nil {
			slog.Error("sync worker wait failed", "shard", i, "error", err)
		}
	}

	if err := partManager.Wait(waitCtx); err != nil {
		slog.Error("partition manager wait failed", "error", err)
	}
	if clickhouseJanitor != nil {
		clickhouseJanitor.Wait()
	}

	cancel()

	if streamTrimmer != nil {
		streamTrimmer.Close()
		streamTrimmer.Wait()
	}

	for i, redisClient := range redisShards {
		if err := redisClient.Close(); err != nil {
			slog.Error("failed to close redis shard", "shard", i, "error", err)
		}
	}
	slog.Info("processor shutdown complete")
}
