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

	"github.com/bidshard/ad-event-processor/internal/clickhouse/migrate"
	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/controlplane"
	"github.com/bidshard/ad-event-processor/internal/database"
	"github.com/bidshard/ad-event-processor/internal/dedup"
	"github.com/bidshard/ad-event-processor/internal/domain"
	db "github.com/bidshard/ad-event-processor/internal/domain/db"
	"github.com/bidshard/ad-event-processor/internal/fraud"
	"github.com/bidshard/ad-event-processor/internal/ingestion"
	"github.com/bidshard/ad-event-processor/internal/licensing"
	"github.com/bidshard/ad-event-processor/internal/metrics"
	"github.com/bidshard/ad-event-processor/internal/notify"
	"github.com/bidshard/ad-event-processor/internal/postback"
	"github.com/bidshard/ad-event-processor/pkg/lifecycle"
	"github.com/bidshard/ad-event-processor/pkg/logger"
	"github.com/bidshard/ad-event-processor/pkg/pgfailover"
	"github.com/bidshard/ad-event-processor/pkg/piihash"
	rpclient "github.com/bidshard/ad-event-processor/pkg/regionproxy/client"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
)

func main() {
	if len(os.Args) > 2 && os.Args[1] == "--health-probe" {
		if !lifecycle.RunHealthProbe(os.Args[2]) {
			os.Exit(1)
		}
		os.Exit(0)
	}
	if licensing.MaybeRunGuardWatchdogCLI(os.Args) {
		return
	}

	slogLogger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(slogLogger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

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
	})

	consumerCtx, consumerCancel := context.WithCancel(context.Background())
	defer consumerCancel()

	syncCtx, syncCancel := context.WithCancel(context.Background())
	defer syncCancel()

	pool, err := database.Connect(ctx, string(cfg.DBDSN), cfg.DBProcessorMaxConns, cfg.DBMinConns)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	settlePool, err := database.Connect(ctx, string(cfg.DBDSN), cfg.PgPoolSettleConns(cfg.SettlementLaneCount()), 1)
	if err != nil {
		slog.Error("failed to connect settlement pool", "error", err)
		os.Exit(1)
	}
	defer settlePool.Close()

	procPgGate := ingestion.NewProcessorPgGate(cfg.ProcessorPGGateSlots, cfg.PgPoolSettleConns(cfg.SettlementLaneCount()))
	procChGate := ingestion.NewProcessorChGate(cfg.ProcessorCHGateSlots, cfg.CHMaxConns)

	queries := db.New(pool)
	settleQueries := db.New(settlePool)
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

	chEnabled := cfg.ClickHouseEnabled()
	var chConn driver.Conn
	var chStore *ingestion.ClickHouseStore
	var chJanitor *database.CHPartitionJanitor
	if chEnabled {
		var err error
		chConn, err = database.ConnectClickHouse(ctx, string(cfg.CHDSN))
		if err != nil {
			slog.Error("failed to connect to clickhouse", "error", err)
			os.Exit(1)
		}
		defer func() { _ = chConn.Close() }()

		if err := migrate.ApplyClickHouseMigrations(ctx, chConn); err != nil {
			slog.Error("failed to apply clickhouse migrations", "error", err)
			os.Exit(1)
		}
	} else {
		slog.Info("clickhouse consumer disabled", "ch_consumer", "disabled")
	}

	var notifierClient *controlplane.NotifierClient
	if cfg.NotifierAPIEnabled() {
		api, closeNotifier, notifierErr := notify.OpenAPI(ctx, cfg)
		if notifierErr != nil {
			slog.Warn("notifier module unavailable for ops alerts", "error", notifierErr)
		} else if api != nil {
			notifierClient = controlplane.NewNotifierClientFromAPI(api)
			if closeNotifier != nil {
				defer closeNotifier()
			}
		}
	}
	opsAlerter := controlplane.NewOpsAlerter(notifierClient, cfg)
	var onEmergencyDrop database.EmergencyDropAlerter
	if chEnabled && opsAlerter != nil && cfg.CHEmergencyDropPercent > 0 {
		threshold := cfg.CHEmergencyDropPercent
		onEmergencyDrop = func(table, partition string, diskPct float64) {
			opsAlerter.AlertCHEmergencyDrop(ctx, table, partition, diskPct, threshold)
		}
	}
	if chEnabled && chConn != nil && cfg.CHJanitorEnabled {
		intervalH := cfg.CHJanitorIntervalH
		if intervalH <= 0 {
			intervalH = 24
		}
		dealDays := cfg.CHRetentionDaysRtbDealOutcomes
		if dealDays <= 0 {
			dealDays = 90
		}
		exchangeDays := cfg.CHRetentionDaysRtbExchangeLog
		if exchangeDays <= 0 {
			exchangeDays = 30
		}
		chJanitor = database.NewCHPartitionJanitor(chConn, database.CHJanitorOptions{
			RetentionDays: cfg.CHRawRetentionDays,
			ExtraTables: []database.CHTableRetention{
				{Table: "rtb_deal_outcomes", Days: dealDays},
				{Table: "rtb_exchange_log", Days: exchangeDays},
			},
			EmergencyDropPercent:     cfg.CHEmergencyDropPercent,
			RecompressPartsThreshold: cfg.CHRecompressPartsThreshold,
			OffPeakStartHourUTC:      cfg.CHRecompressOffPeakStartUTC,
			OffPeakEndHourUTC:        cfg.CHRecompressOffPeakEndUTC,
			OnEmergencyDrop:          onEmergencyDrop,
		})
		chJanitor.StartBackground(ctx, time.Duration(intervalH)*time.Hour)
	}

	var rdbs []redis.UniversalClient
	rdbs, _, err = database.ConnectRedisShards(ctx, cfg, database.RedisShardOptions{
		PoolSize: cfg.RedisPoolSize,
	})
	if err != nil {
		slog.Error("failed to connect to redis shards", "error", err)
		os.Exit(1)
	}

	streamTrimmer := ingestion.NewRedisStreamTrimmer(ingestion.RedisStreamTrimmerConfig{
		Rdbs:         rdbs,
		Streams:      []string{cfg.RedisStreamName, cfg.FraudStreamName},
		MaxLen:       cfg.StreamMaxLen,
		TrimInterval: time.Duration(cfg.RedisStreamTrimIntervalMs) * time.Millisecond,
	})
	streamTrimmer.Start(consumerCtx)

	pgStore := ingestion.NewPostgresStoreWithGate(settleQueries, time.Duration(cfg.WriteTimeoutMs)*time.Millisecond, procPgGate)
	piiHasher, piiErr := piihash.NewFromConfig(cfg)
	if piiErr != nil {
		slog.Error("failed to initialize PII hasher", "error", piiErr)
		os.Exit(1)
	}
	if cfg.EventsHashIPAtInsert {
		pgStore.SetPIIHashAtInsert(piiHasher)
		slog.Info("postgres events IP hash-at-insert enabled")
	}
	if chEnabled && chConn != nil {
		spoolCfg := ingestion.CHCfgFromConfig(cfg.CHSpoolSegmentMB, cfg.CHSpoolMaxSegments)
		chStore = ingestion.NewClickHouseStore(chConn, time.Duration(cfg.WriteTimeoutMs)*time.Millisecond, cfg.CHSpoolDir, spoolCfg, procChGate)
		chStore.SetPIIHasher(piiHasher)
		if err := chStore.RecoverSpool(ctx); err != nil {
			slog.Error("failed to recover clickhouse spool", "error", err)
			os.Exit(1)
		}
	}

	settleStore := domain.EventStore(pgStore)
	if chStore != nil {
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
	if cfg.FraudScoringEnabled() {
		snap, snapErr := licensing.LoadDeploymentSnapshot(ctx, pool)
		if snapErr == nil && snap.ModuleAllowed(func(f licensing.FeatureSet) bool { return f.MlFraudBoostEnabled() }) {
			var err error
			fraudScorer, err = fraud.NewLGBMScorer(cfg.FraudScoring.ModelPath)
			if err != nil {
				slog.Error("failed to initialize fraud scorer for processor micro-batching", "error", err, "path", cfg.FraudScoring.ModelPath)
				os.Exit(1)
			}
			slog.Info("initialized fraud scorer for processor micro-batching", "path", cfg.FraudScoring.ModelPath)
		} else {
			slog.Info("ml_fraud_boost not licensed; processor fraud micro-batching disabled")
		}
	}

	campaignRepo := ingestion.NewCampaignRepoWithDB(pool, queries)
	campaignRepo.ConfigureAuditLedgerFlush(cfg.AuditLedgerFlushSampleMask)
	customerRepo := ingestion.NewCustomerRepoWithDB(pool, queries)
	dedupAdapter := dedup.NewAdapter(pool, cfg.RegionCode, dedup.LoadRoutingEpoch(ctx, pool))

	var ingestPgFailover *pgfailover.IngestRuntime
	ingestPgFailover = pgfailover.StartIngestSubscribers(ctx, rdbs, pgfailover.IngestSubscriberConfig{
		MaxConns: cfg.DBProcessorMaxConns,
		MinConns: cfg.DBMinConns,
		Interval: time.Duration(cfg.PgFailoverPollMs) * time.Millisecond,
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
		settleNew, connectErr := database.Connect(ctx, dsn, cfg.PgPoolSettleConns(cfg.SettlementLaneCount()), 1)
		if connectErr != nil {
			slog.Warn("processor pg failover settle pool reconnect failed", "error", connectErr)
			return
		}
		oldSettle := settlePool
		settlePool = settleNew
		settleQueries = db.New(settleNew)
		pgStore.SetQuerier(settleQueries)
		if postbackEnqueuer != nil {
			postbackEnqueuer.SetStore(settleQueries)
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
		weightCtrl = ingestion.NewProcessorWeightController(ingestion.ProcessorWeightConfigFromApp(cfg), procPgGate, nil)
		weightCtrl.Start(ctx)
		slog.Info("processor weight scheduling enabled", "node_id", cfg.NodeID)
	}

	var pgSettlementWorkers []*ingestion.SettlementWorker
	var chConsumers []*ingestion.StreamConsumer
	var brokerConsumers []*ingestion.BrokerStreamConsumer
	var brokerCHGroup *BrokerConsumerGroup
	var brokerFraudGroup *BrokerConsumerGroup
	var brokerReconcile *ingestion.BrokerReconcileWorker
	var budgetDeltaConsumer *controlplane.BudgetDeltaConsumer
	var syncWorkers []*ingestion.SyncWorker
	var fraudMicrobatcher *fraud.MicroBatcher
	if fraudScorer != nil && len(rdbs) > 0 {
		fraudMicrobatcher = fraud.NewMicroBatcher(rdbs, fraudScorer, cfg.CampaignUpdateChannel)
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

	for i, rdb := range rdbs {
		shardID := fmt.Sprintf("shard_%d", i)

		sw := ingestion.NewSyncWorker(rdb, campaignRepo, customerRepo, time.Duration(cfg.BudgetSyncIntervalMs)*time.Millisecond, time.Duration(cfg.LedgerBatchFlushMs)*time.Millisecond, procPgGate, 0)
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
			settleFlush := time.Duration(cfg.SettlementFlushMs) * time.Millisecond
			settleW := ingestion.NewSettlementWorker(
				settleStore,
				rdb,
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
			segmentHandler := ingestion.NewSegmentConversionHandler(campaignRepo, queries, []redis.UniversalClient{rdb}, piiHasher)
			settleW.SetOnMessageProcessed(segmentHandler.Handle)
			pgSettlementWorkers = append(pgSettlementWorkers, settleW)
			settleW.Start(consumerCtx)
		} else {
			slog.Info("processor: Redis SettlementWorker disabled (CH_INGEST_SOURCE=broker)")
		}

		if chStore != nil && !cfg.BrokerPrimaryCH() {
			cc := ingestion.NewStreamConsumer(
				chStore,
				rdb,
				cfg.RedisStreamName,
				cfg.RedisGroupName+"_ch",
				cfg.RedisConsumerID+"_"+shardID,
				cfg.CHBatchSize,
				cfg.ProcessorCHStreamWorkers(),
				time.Duration(cfg.CHFlushIntervalMs)*time.Millisecond,
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
		} else if chStore != nil && cfg.BrokerPrimaryCH() {
			slog.Info("processor: Redis _ch StreamConsumer disabled (CH_INGEST_SOURCE=broker)")
		}

		if chStore != nil && !cfg.BrokerPrimaryCH() {
			fc := ingestion.NewStreamConsumer(
				chStore,
				rdb,
				cfg.FraudStreamName,
				cfg.RedisGroupName+"_fraud",
				cfg.RedisConsumerID+"_fraud_"+shardID,
				cfg.CHBatchSize,
				cfg.ProcessorCHStreamWorkers(),
				time.Duration(cfg.CHFlushIntervalMs)*time.Millisecond,
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
			rdbs,
			cfg.FraudStreamName,
			cfg.RedisGroupName+"_fraud",
			cfg.FraudConsumerLagSec,
			2*time.Second,
		)
	}

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

		for p := 0; p < partCount; p++ {
			pgBrokerCfg := brokerBase
			pgBrokerCfg.Partition = uint16(p)
			pgBrokerCfg.Group = cfg.RedisGroupName + "_pg_broker"
			pgBroker := ingestion.NewBrokerStreamConsumer(pgStoreForBroker, pgBrokerCfg, writeTimeout, retryInit, retryMax, cfg.MaxRetries)
			pgBroker.SetDedupAdapter(dedupAdapter)
			pgBroker.SetLogger(appLogger)
			brokerConsumers = append(brokerConsumers, pgBroker)
			pgBroker.Start(consumerCtx)
		}

		if chStore != nil {
			chGroupCfg := BrokerConsumerGroupConfig{
				BrokerAddr:     cfg.Broker.URL,
				RedisURL:       brokerRedisURL,
				Topic:          cfg.Broker.Topic,
				Group:          cfg.RedisGroupName + "_ch_broker",
				PartitionCount: partCount,
				BatchSize:      cfg.CHBatchSize,
				FlushInterval:  time.Duration(cfg.CHFlushIntervalMs) * time.Millisecond,
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
			brokerCHGroup, chGrpErr = NewBrokerConsumerGroup(chStore, chGroupCfg, appLogger)
			if chGrpErr != nil {
				slog.Error("failed to create broker consumer group for clickhouse", "error", chGrpErr)
			} else {
				brokerCHGroup.Start(consumerCtx)
			}
		}

		if chStore != nil && cfg.BrokerPrimaryCH() {
			slog.Info("processor: Redis _fraud StreamConsumer disabled (CH_INGEST_SOURCE=broker)")
			fraudGroupCfg := BrokerConsumerGroupConfig{
				BrokerAddr:     cfg.Broker.URL,
				RedisURL:       brokerRedisURL,
				Topic:          cfg.Broker.FraudTopic,
				Group:          cfg.RedisGroupName + "_fraud_broker",
				PartitionCount: partCount,
				BatchSize:      cfg.CHBatchSize,
				FlushInterval:  time.Duration(cfg.CHFlushIntervalMs) * time.Millisecond,
				MaxBytes:       uint32(cfg.Broker.MaxBytes),
				Timeout:        time.Duration(cfg.Broker.TimeoutMs) * time.Millisecond,
				DataDir:        cfg.Logger.Dir + "/offsets/fraud",
				ShadowMode:     cfg.Broker.ShadowMode,
			}
			var fraudGrpErr error
			brokerFraudGroup, fraudGrpErr = NewBrokerConsumerGroup(chStore, fraudGroupCfg, appLogger)
			if fraudGrpErr != nil {
				slog.Error("failed to create broker consumer group for fraud clickhouse", "error", fraudGrpErr)
			} else {
				brokerFraudGroup.Start(consumerCtx)
			}
		}

		if len(rdbs) > 0 && !cfg.BrokerPrimaryCH() {
			brokerReconcile = ingestion.NewBrokerReconcileWorker(ingestion.BrokerReconcileConfig{
				BrokerAddr:          cfg.Broker.URL,
				BrokerRedis:         brokerRedisURL,
				Topic:               cfg.Broker.Topic,
				PartitionCount:      partCount,
				BrokerGroup:         cfg.RedisGroupName + "_pg_broker",
				StreamName:          cfg.RedisStreamName,
				Interval:            time.Duration(cfg.Broker.ReconcileIntervalMs) * time.Millisecond,
				DivergenceThreshold: cfg.Broker.DivergenceThreshold,
			}, rdbs)
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
		brokerRedisURL := cfg.Broker.RedisURL
		if brokerRedisURL == "" && len(cfg.RedisAddrs) > 0 {
			brokerRedisURL = database.BrokerRedisURL(cfg.RedisAddrs, string(cfg.RedisPassword))
		}
		budgetDeltaConsumer = controlplane.NewBudgetDeltaConsumer(
			domain.NewBudgetDeltaAggregator(),
			domain.BrokerConsumerConfig{
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

	mux := http.NewServeMux()
	mux.Handle("GET /metrics", promhttp.Handler())
	live := &lifecycle.Liveness{}
	ready := &lifecycle.ReadinessProbe{}
	ready.StartBackground(ctx, 2*time.Second, func(probeCtx context.Context) bool {
		if err := pool.Ping(probeCtx); err != nil {
			return false
		}
		if chConn != nil {
			if err := chConn.Ping(probeCtx); err != nil {
				return false
			}
		}
		for i, rdb := range rdbs {
			if err := rdb.Ping(probeCtx).Err(); err != nil {
				return false
			}
			if cfg.RedisStreamName != "" {
				if n, err := rdb.XLen(probeCtx, cfg.RedisStreamName).Result(); err == nil {
					metrics.ProcessorStreamXLen.WithLabelValues(strconv.Itoa(i)).Set(float64(n))
				}
			}
		}
		if chStore != nil {
			if spool := chStore.Spool(); spool != nil {
				seg := spool.SegmentCount()
				metrics.CHSpoolSegments.Set(float64(seg))
				if seg > cfg.CHSpoolMaxSegments {
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
	if chStore != nil {
		_ = chStore.Close()
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
	if chJanitor != nil {
		chJanitor.Wait()
	}

	cancel()

	if streamTrimmer != nil {
		streamTrimmer.Close()
		streamTrimmer.Wait()
	}

	for i, rdb := range rdbs {
		if err := rdb.Close(); err != nil {
			slog.Error("failed to close redis shard", "shard", i, "error", err)
		}
	}
	slog.Info("processor shutdown complete")
}
