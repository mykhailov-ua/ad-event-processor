package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bidshard/ad-event-processor/internal/clickhouse/migrate"
	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/database"
	db "github.com/bidshard/ad-event-processor/internal/domain/db"
	"github.com/bidshard/ad-event-processor/internal/ingestion"
	"github.com/bidshard/ad-event-processor/internal/licensing"
	"github.com/bidshard/ad-event-processor/internal/metrics"
	"github.com/bidshard/ad-event-processor/internal/rtb"
	"github.com/bidshard/ad-event-processor/pkg/lifecycle"
	"github.com/bidshard/ad-event-processor/pkg/logger"
	"github.com/bidshard/ad-event-processor/pkg/netaddr"
	"github.com/bidshard/ad-event-processor/pkg/pgfailover"
	"github.com/bidshard/ad-event-processor/pkg/piihash"
	"github.com/bidshard/ad-event-processor/pkg/runtimeautotune"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/panjf2000/gnet/v2"
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

	slogLogger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	slog.SetDefault(slogLogger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}
	runtimeautotune.Apply(cfg)

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

	pool, err := database.Connect(ctx, string(cfg.DBDSN), cfg.DBTrackerMaxConns, cfg.DBMinConns)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	queries := db.New(pool)
	registry := ingestion.NewRegistry(queries)
	registry.SetPool(pool)
	if cfg.CampaignReplicaPath != "" {
		registry.SetReplicaPath(cfg.CampaignReplicaPath)
	}
	if replicaCount, err := registry.BootstrapFromReplica(); err != nil {
		slog.Warn("campaign replica bootstrap failed", "error", err)
	} else if replicaCount > 0 {
		slog.Info("campaign registry preloaded from replica", "campaigns", replicaCount)
	}
	count, err := registry.Sync(ctx)
	if err != nil {
		slog.Warn("initial campaign registry sync failed", "error", err)
	} else {
		slog.Info("campaign registry loaded", "campaigns", count)
	}
	registry.StartSync(ctx, time.Duration(cfg.RegistrySyncIntervalMs)*time.Millisecond)

	if config.LicenseRequiredFromEnv() {
		recheckInterval := 5 * time.Minute
		if d, parseErr := time.ParseDuration(config.LicenseFileRecheckInterval()); parseErr == nil && d > 0 {
			recheckInterval = d
		}
		licensePath := config.LicensePathFromEnv()
		registry.StartLicenseRecheck(ctx, ingestion.RegistryLicenseConfig{
			Required: true,
			Path:     licensePath,
			Interval: recheckInterval,
		})
		slog.Info("license file recheck enabled", "path", licensePath, "interval", recheckInterval.String())
	}

	var rdbs []redis.UniversalClient
	var breakers []*database.RedisBreaker
	rdbs, breakers, err = database.ConnectRedisShards(ctx, cfg, database.RedisShardOptions{
		PoolSize:         cfg.RedisPoolSize,
		FilterTimeoutMs:  cfg.FilterTimeoutMs,
		StickyPinWorkers: cfg.MaxWorkers,
	})
	if err != nil {
		slog.Error("failed to connect to redis shards", "error", err)
		os.Exit(1)
	}
	database.StartRedisPoolStatsReporter(ctx, rdbs, 15*time.Second)
	if rdbs[0] == nil {
		slog.Warn("redis shard 0 not connected; running in degraded mode",
			"replica", cfg.CampaignReplicaPath,
			"broker_fallback", cfg.CampaignUpdateBrokerFallback,
		)
	}

	channel := cfg.CampaignUpdateChannel
	if channel == "" {
		channel = "campaigns:update"
	}
	campaignRepo := ingestion.NewCampaignRepo(queries)
	sharder := ingestion.NewStaticSlotSharder(len(rdbs))
	if version, loadErr := ingestion.LoadActiveSlotMap(ctx, pool, sharder, len(rdbs)); loadErr != nil {
		slog.Warn("slot map load failed, using modulo fallback", "error", loadErr)
	} else {
		slog.Info("slot map loaded at startup", "version", version)
	}

	slotMapWatcher := ingestion.NewSlotMapWatcher(ingestion.SlotMapWatcherConfig{
		Pool:           pool,
		Sharder:        sharder,
		NumShards:      len(rdbs),
		PollInterval:   time.Duration(cfg.SlotMapPollIntervalMs) * time.Millisecond,
		BrokerURL:      cfg.Broker.URL,
		BrokerRedisURL: cfg.Broker.RedisURL,
		BrokerTopic:    cfg.SlotMapReloadTopic,
		BrokerTimeout:  time.Duration(cfg.Broker.TimeoutMs) * time.Millisecond,
	})
	go slotMapWatcher.Start(ctx)

	budgetWarmer := ingestion.NewBudgetCacheWarmer(rdbs, sharder)
	registry.SetBudgetWarmer(budgetWarmer)
	if warmed, err := budgetWarmer.WarmFromRegistry(ctx, registry); err != nil {
		slog.Error("initial budget cache warm failed", "error", err)
	} else {
		slog.Info("budget cache warmed", "keys_inserted", warmed)
	}

	registry.ConfigureStaleMode(time.Duration(cfg.RegistryStaleTTLSec) * time.Second)
	registry.StartWatchShards(ctx, rdbs, channel)
	registry.StartEpochPoll(ctx, rdbs, time.Duration(cfg.RegistryPollMs)*time.Millisecond)

	if cfg.CampaignUpdateBrokerFallback && cfg.Broker.URL != "" {
		topic := cfg.CampaignUpdateBrokerTopic
		if topic == "" {
			topic = ingestion.DefaultCampaignUpdateBrokerTopic
		}
		cuWatcher := ingestion.NewCampaignUpdateWatcher(ingestion.CampaignUpdateWatcherConfig{
			Registry:       registry,
			BrokerURL:      cfg.Broker.URL,
			BrokerRedisURL: cfg.Broker.RedisURL,
			BrokerTopic:    topic,
			BrokerTimeout:  time.Duration(cfg.Broker.TimeoutMs) * time.Millisecond,
		})
		go cuWatcher.Start(ctx)
		slog.Info("campaign update broker fallback enabled", "topic", topic)
	}

	consentChannel := cfg.ConsentUpdateChannel
	if consentChannel == "" {
		consentChannel = ingestion.ConsentDefaultUpdateChannel
	}
	consentRdb := firstConnectedRedis(rdbs)
	consentStore := ingestion.NewConsentStore(consentRdb)
	if consentRdb != nil {
		consentStore.StartWatch(ctx, consentRdb, consentChannel)
	} else {
		slog.Warn("consent pub/sub disabled: no redis shard available")
	}

	var geoProvider ingestion.GeoProvider
	geoProvider, err = ingestion.NewMaxMindProvider(cfg.GeoIP.DBPath)
	if err != nil {
		if cfg.Env == "prod" || cfg.Env == "production" {
			slog.Error("FATAL: MaxMind DB load failed in production", "error", err)
			os.Exit(1)
		}
		slog.Warn("MaxMind DB load failed, using mock geo provider (development only)", "error", err)
		geoProvider = &ingestion.MockGeoProvider{}
	}
	defer func() { _ = geoProvider.Close() }()

	if mm, ok := geoProvider.(*ingestion.MaxMindProvider); ok {
		metrics.GeoProviderStatus.Set(1)
		if err := mm.ReloadASN(cfg.GeoIP.ASNDBPath); err != nil {
			slog.Warn("geoip asn db load failed; hot DC ASN lookup disabled", "path", cfg.GeoIP.ASNDBPath, "error", err)
		}
		watcherInterval := time.Duration(cfg.GeoIP.WatcherIntervalSec) * time.Second
		go ingestion.NewGeoIPWatcher(mm, cfg.GeoIP.DBPath, cfg.GeoIP.ASNDBPath, watcherInterval).Start(ctx)
		slog.Info("geoip hot-reload watcher started", "country_path", cfg.GeoIP.DBPath, "asn_path", cfg.GeoIP.ASNDBPath, "interval", watcherInterval)
	} else {
		metrics.GeoProviderStatus.Set(0)
	}

	geoFilter := ingestion.NewGeoFilter(geoProvider, registry)
	scheduleFilter := ingestion.NewScheduleFilter(registry)
	fraudFilter := ingestion.NewFraudFilter(geoProvider)
	if cfg.DCASNHotEnabled {
		dcASNTable := ingestion.NewDCASNTable()
		if loader := ingestion.NewDCASNFeedLoader(cfg, dcASNTable); loader != nil {
			go loader.Start(ctx)
			slog.Info("dc asn hot loader started", "dir", cfg.DCASNFeedDir, "refresh", cfg.DCASNFeedRefresh)
		}
		if lookup, ok := geoProvider.(ingestion.ASNLookup); ok {
			fraudFilter.ConfigureDCASN(dcASNTable, lookup, cfg.DCASNSampleMask)
		}
	}
	var tcpMSSFilter ingestion.EventFilter
	if cfg.TCPMSSAnomalyEnabled {
		tcpMSSFilter = ingestion.NewTCPMSSFilter(cfg.TCPMSSAnomalyMinByte)
	}
	var residentialProxyFilter ingestion.EventFilter
	if cfg.ResidentialProxyHotEnabled {
		proxyRing := ingestion.NewResidentialProxyRing()
		proxyRing.SetPolicy(ingestion.ResidentialProxyPolicyFromEnv())
		proxyRing.SetWindow(cfg.ResidentialProxyWindow)
		residentialProxyFilter = ingestion.NewResidentialProxyFilter(proxyRing)
		slog.Info("residential proxy hot signal enabled", "window", cfg.ResidentialProxyWindow)
	}

	settingsWatcher := ingestion.NewSettingsWatcher(rdbs, cfg)
	settingsWatcher.SetPGFallback(ingestion.SettingsPGSync(pool), registry.IsStaleMode)
	deviceFilter := ingestion.NewDeviceFilter(settingsWatcher)
	deviceFilter.SetOSFingerprintEnabled(cfg.OSFingerprintMismatchEnabled)
	go settingsWatcher.Start(ctx, time.Second)

	breakerFilter := ingestion.NewEmergencyBreakerFilter(settingsWatcher)
	consentFilter := ingestion.NewConsentFilter(registry, consentStore)

	streamTrimmer := ingestion.NewRedisStreamTrimmer(ingestion.RedisStreamTrimmerConfig{
		Rdbs:         rdbs,
		Streams:      []string{cfg.RedisStreamName, cfg.FraudStreamName},
		MaxLen:       cfg.StreamMaxLen,
		TrimInterval: time.Duration(cfg.RedisStreamTrimIntervalMs) * time.Millisecond,
	})
	streamTrimmer.Start(ctx)

	trackerStreamName := cfg.RedisStreamName
	if cfg.BrokerEnabled() && cfg.BrokerPrimaryCH() {
		trackerStreamName = "fcap:ignored"
	}

	if err := ingestion.InitUnifiedFilterLua(); err != nil {
		slog.Error("unified-filter lua init failed", "error", err)
		os.Exit(1)
	}

	unifiedFilter := ingestion.NewUnifiedFilter(
		rdbs,
		sharder,
		registry,
		campaignRepo,
		0,
		time.Duration(cfg.RateLimitWindowMs)*time.Millisecond,
		time.Duration(cfg.DuplicateTTLSec)*time.Second,
		time.Duration(cfg.IdempotencyTTLHrs)*time.Hour,
		cfg.ClickAmount,
		cfg.ImpressionAmount,
		trackerStreamName,
		cfg.StreamMaxLen,
	)
	unifiedFilter.SetFilterEvalPinWorkers(cfg.MaxWorkers)
	unifiedFilter.SetShardBreakers(breakers)
	unifiedFilter.SetSettingsWatcher(settingsWatcher)
	if err := unifiedFilter.PreloadScripts(ctx); err != nil {
		slog.Error("failed to preload redis lua scripts on all shards", "error", err)
		os.Exit(1)
	}
	unifiedFilter.AttachReconnectPreload()
	unifiedFilter.StartScriptPreheater(ctx, 30*time.Second)
	unifiedFilter.SetTTCMin(time.Duration(cfg.TTCMinMs) * time.Millisecond)
	unifiedFilter.SetTTCFailClosed(cfg.TTCFailClosed)
	if cfg.TTCMinMs > 0 {
		unifiedFilter.SetLocalTTCCache(ingestion.NewLocalTTCCache())
	}
	unifiedFilter.SetRoughPacingGate(ingestion.NewRoughPacingGate())
	unifiedFilter.SetMetricsSampleMask(cfg.MetricsHistogramSampleMask)
	unifiedFilter.SetQuotaConfig(cfg.QuotaMode, cfg.QuotaChunkSize, cfg.QuotaRefillThresholdPct)
	unifiedFilter.SetLuaFastPathEnabled(cfg.LuaFastPathEnabled)
	unifiedFilter.SetRegionCode(cfg.RegionCode)
	unifiedFilter.SetPGFallbackAllowed(cfg.TrackerPGFallback)

	var localQuantaLedger *ingestion.LocalQuantaLedger
	var quotaRefillWorker *ingestion.QuotaRefillWorker
	var budgetDeltaPublisher *ingestion.BudgetDeltaPublisher
	var localQuantaFlusher *ingestion.LocalQuantaFlusher
	var localQuantaStream *ingestion.LocalQuantaStreamPublisher
	if cfg.LocalQuotaMode == "shadow" || cfg.LocalQuotaMode == "live" {
		localQuantaLedger = ingestion.NewLocalQuantaLedger()
		localQuantaStrict := ingestion.NewLocalQuantaStrict(cfg.QuotaStrictThresholdMicro, cfg.QuotaStrictExitMicro)
		chunkSize := cfg.QuotaChunkSize
		if chunkSize <= 0 {
			chunkSize = 5_000_000
		}
		quotaRefillWorker = ingestion.NewQuotaRefillWorker(
			localQuantaLedger,
			rdbs,
			sharder,
			ingestion.QuotaRefillConfig{
				BaseChunkMicro: chunkSize,
				ThresholdPct:   cfg.QuotaRefillThresholdPct,
				MaxPerShard:    cfg.LocalQuotaRefillMaxShard,
				FloorMicro:     cfg.QuotaAdaptiveFloorMicro,
				CeilingMicro:   cfg.QuotaAdaptiveCeilingMicro,
				StrictEnter:    cfg.QuotaStrictThresholdMicro,
			},
		)
		brokerRedisURL := cfg.Broker.RedisURL
		if brokerRedisURL == "" && len(cfg.RedisAddrs) > 0 {
			brokerRedisURL = database.BrokerRedisURL(cfg.RedisAddrs, string(cfg.RedisPassword))
		}
		budgetDeltaPublisher = ingestion.NewBudgetDeltaPublisher(ingestion.BudgetDeltaPublisherConfig{
			BrokerAddr: cfg.Broker.URL,
			RedisURL:   brokerRedisURL,
			Topic:      cfg.BudgetDeltaTopic,
			TrackerID:  cfg.RedisConsumerID,
			Timeout:    time.Duration(cfg.Broker.TimeoutMs) * time.Millisecond,
		})
		localQuantaFlusher = ingestion.NewLocalQuantaFlusher(localQuantaLedger, rdbs, sharder, budgetDeltaPublisher)
		idemCache := ingestion.NewLocalClickIdemCache(time.Duration(cfg.IdempotencyTTLHrs) * time.Hour)
		localQuantaStream = ingestion.NewLocalQuantaStreamPublisher(ingestion.LocalQuantaStreamPublisherConfig{
			Rdbs:           rdbs,
			StreamName:     trackerStreamName,
			MaxLen:         cfg.StreamMaxLen,
			IdempotencyTTL: time.Duration(cfg.IdempotencyTTLHrs) * time.Hour,
			IdemCache:      idemCache,
		})
		quotaRefillWorker.SetStrictMode(localQuantaStrict, localQuantaFlusher)
		quotaRefillWorker.SetCampaignRegistry(registry)
		localQuantaFlusher.SetCampaignRegistry(registry)
		ingestion.SetRegistryQuantaFlushHook(func(id uuid.UUID) {
			flushCtx, flushCancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer flushCancel()
			localQuantaFlusher.FlushLocalQuanta(flushCtx, id, ingestion.FlushReasonPause)
		})
		if cfg.Broker.URL != "" {
			recoveryCtx, recoveryCancel := context.WithTimeout(ctx, 10*time.Second)
			deltas, recErr := ingestion.FetchRecoveryDeltas(recoveryCtx, ingestion.BrokerConsumerConfig{
				BrokerAddr: cfg.Broker.URL,
				RedisURL:   brokerRedisURL,
				Topic:      cfg.BudgetDeltaTopic,
				MaxBytes:   uint32(cfg.Broker.MaxBytes),
				Timeout:    time.Duration(cfg.Broker.TimeoutMs) * time.Millisecond,
			}, 0)
			recoveryCancel()
			if recErr != nil {
				slog.Warn("local quanta broker recovery skipped", "error", recErr)
			} else if len(deltas) > 0 {
				quotaRefillWorker.RecoverFromDeltas(deltas)
				slog.Info("local quanta ledger recovered from broker", "campaigns", len(deltas))
			}
		}
		unifiedFilter.SetLocalQuantaDeps(ingestion.LocalQuantaDeps{
			Ledger:    localQuantaLedger,
			Strict:    localQuantaStrict,
			Refill:    quotaRefillWorker,
			Publisher: budgetDeltaPublisher,
			Stream:    localQuantaStream,
		})
		unifiedFilter.SetLocalQuantaMode(cfg.LocalQuotaMode)
		slog.Info("local quanta enabled",
			"mode", cfg.LocalQuotaMode,
			"chunk_size", chunkSize,
			"refill_threshold_pct", cfg.QuotaRefillThresholdPct,
			"strict_enter_micro", cfg.QuotaStrictThresholdMicro,
			"strict_exit_micro", cfg.QuotaStrictExitMicro,
		)
		if cfg.LocalQuotaMode == "live" {
			slog.Info("local quota full-skip ratio: rate(ad_local_quota_full_skip_total[5m]) / rate(ad_local_quota_full_skip_eligible_total[5m])")
		}
	}
	unifiedFilter.SetFilterSlowMs(cfg.FilterSlowMs)
	if cfg.TTCFailClosed {
		slog.Info("TTC fail-closed enabled: clicks without impression timestamp are rejected")
	}
	slog.Info("redis lua scripts preloaded", "shards", len(rdbs))

	creativeStore := ingestion.NewBrandCreativeStore(firstConnectedRedis(rdbs), cfg.FilterTimeoutMs)
	placementBL := ingestion.NewPlacementBlacklistFilter(rdbs)
	unifiedFilter.SetPlacementBlacklistFilter(placementBL)
	unifiedFilter.SetFraudBlacklistFilter(ingestion.NewFraudBlacklistFilter(rdbs))
	unifiedFilter.SetIngressRPDHandledExternally(true)
	licenseFilter := ingestion.NewLicenseFilter(registry)
	licenseRPSFilter := ingestion.NewLicenseRPSFilter(registry)
	entitlementsFilter := ingestion.NewEntitlementsFilter(registry, sharder, rdbs)
	entitlementsFilter.SetRegionCode(uint8(cfg.RegionCode))
	piiHasher, piiErr := piihash.NewFromConfig(cfg)
	if piiErr != nil {
		slog.Error("failed to initialize PII hasher for segment filter", "error", piiErr)
		os.Exit(1)
	}
	vppFilter := ingestion.NewVPPFilter(registry, settingsWatcher)
	segmentFilter := ingestion.NewSegmentFilter(rdbs, registry, piiHasher)
	filterEngine := ingestion.NewFilterEngine(time.Duration(cfg.FilterTimeoutMs)*time.Millisecond, licenseFilter, licenseRPSFilter, breakerFilter, geoFilter, scheduleFilter, segmentFilter, vppFilter, fraudFilter, residentialProxyFilter, tcpMSSFilter, deviceFilter, consentFilter, entitlementsFilter, unifiedFilter)
	filterEngine.SetSettingsWatcher(settingsWatcher)

	var rtbCatalog *ingestion.RtbCatalog
	var rtbHybrid *ingestion.HybridBalancer
	var rtbReconcile *ingestion.RtbBudgetReconcileWorker
	rtbBudgetSync := ingestion.RtbBudgetSync{
		Authority: ingestion.BudgetAuthorityShadow,
		Redis:     rdbs,
		Sharder:   sharder,
	}
	if cfg.RtbEnabled() {
		rtb.SetMetricsEnabled(true)
		rtbStore := rtb.NewBudgetStore()
		rtbBudgetSync.Authority = ingestion.BudgetAuthorityFromConfig(cfg)
		rtbCatalog = ingestion.NewRtbCatalog(rtbStore, rtbBudgetSync.Authority)
		rtbHybrid = ingestion.NewHybridBalancer(len(rdbs), ingestion.HybridMaxRPSFromConfig(cfg))
		if cfg.RtbClearingMode == "first" {
			rtbCatalog.SetClearingMode(rtb.ClearingFirstPrice)
		}
		if cfg.RtbTargetingIndexEnabled() {
			rtbCatalog.Registry().SetTargetingIndexEnabled(true)
		}
		ingestion.StartRtbCatalogSync(ctx, registry, rtbCatalog, cfg, rtbHybrid, rtbBudgetSync, settingsWatcher, time.Duration(cfg.RegistrySyncIntervalMs)*time.Millisecond)
		if err := ingestion.ReloadRtbDeals(ctx, queries, rtbCatalog); err != nil {
			slog.Warn("initial rtb deals load failed", "error", err)
		} else {
			slog.Info("rtb deals loaded", "count", rtbCatalog.DealCount())
		}
		ingestion.StartRtbCatalogReloadWatch(ctx, queries, firstConnectedRedis(rdbs), ingestion.RtbCatalogReloadChannel(cfg), registry, rtbCatalog, cfg, rtbHybrid, rtbBudgetSync, settingsWatcher)
		dealFloorCache := ingestion.NewDealFloorCache(firstConnectedRedis(rdbs))
		rtbCatalog.SetDealFloors(dealFloorCache)
		ingestion.StartDealFloorRefresh(ctx, dealFloorCache, rtbCatalog, time.Duration(cfg.DealFloorRefreshIntervalMs)*time.Millisecond)
		if allow, err := ingestion.LoadSupplyChainAllowlist(ctx, queries); err == nil {
			rtbCatalog.SetSupplyChainAllowlist(allow)
		} else {
			slog.Warn("initial supply chain allowlist load failed", "error", err)
		}
		var rtbBudgetMirror *ingestion.RtbBudgetMirrorWriter
		if rtbBudgetSync.Authority == ingestion.BudgetAuthorityRTB {
			rtbBudgetMirror = ingestion.NewRtbBudgetMirrorWriter(rtbCatalog, registry, rdbs, sharder)
			defer func() {
				if rtbBudgetMirror != nil {
					rtbBudgetMirror.Close()
				}
			}()
		}
		_ = ingestion.NewRtbAuthorityController(cfg, settingsWatcher, unifiedFilter, rtbCatalog, &rtbBudgetSync)
		rtbReconcile = ingestion.NewRtbBudgetReconcileWorker(
			ingestion.RtbBudgetReconcileConfig{
				Interval:            time.Duration(cfg.RtbReconcileIntervalMs) * time.Millisecond,
				DivergenceThreshold: cfg.RtbBudgetDivergenceMicro,
				SampleSize:          cfg.RtbReconcileSampleSize,
			},
			registry,
			rtbCatalog,
			rdbs,
			sharder,
		)
		rtbReconcile.Start(ctx)
		if snapPath := cfg.RtbSnapshotPath; snapPath != "" {
			if err := rtbCatalog.Registry().StartPersistence(ctx, snapPath, time.Minute); err != nil {
				slog.Warn("rtb snapshot persistence disabled", "error", err)
			} else {
				slog.Info("rtb snapshot persistence enabled", "path", snapPath)
			}
		}
		slog.Info("rtb catalog enabled",
			"mode", cfg.RtbMode,
			"budget_authority", cfg.RtbBudgetAuthority,
			"targeting_index", cfg.RtbTargetingIndexEnabled(),
		)
		if cfg.ClickHouseEnabled() {
			chCtx, chCancel := context.WithTimeout(ctx, 15*time.Second)
			chConn, chErr := database.ConnectClickHouse(chCtx, string(cfg.CHDSN))
			chCancel()
			if chErr != nil {
				slog.Warn("rtb clickhouse writers disabled", "error", chErr)
			} else {
				migCtx, migCancel := context.WithTimeout(ctx, 30*time.Second)
				if migErr := migrate.ApplyClickHouseMigrations(migCtx, chConn); migErr != nil {
					slog.Warn("rtb clickhouse migrate failed", "error", migErr)
				}
				migCancel()
				flush := time.Duration(cfg.RtbDealOutcomeFlushMs) * time.Millisecond
				dealWriter := ingestion.NewRtbDealOutcomeWriter(chConn, flush)
				exchangeWriter := ingestion.NewRtbExchangeLogWriter(chConn, flush)
				ingestion.SetRtbDealOutcomeWriter(dealWriter)
				ingestion.SetRtbExchangeLogWriter(exchangeWriter)
				defer func() {
					dealWriter.Close()
					exchangeWriter.Close()
					_ = chConn.Close()
				}()
				slog.Info("rtb clickhouse writers enabled", "flush_ms", cfg.RtbDealOutcomeFlushMs)
			}
		}
	}

	gnetHandler := ingestion.NewAdsPacketHandler(cfg, registry, filterEngine, pool, rdbs, sharder, cfg.FraudStreamName, creativeStore)

	if cfg.PgFailoverEnabled {
		ingestPgFailover := pgfailover.StartIngestSubscribers(ctx, rdbs, pgfailover.IngestSubscriberConfig{
			MaxConns: cfg.DBTrackerMaxConns,
			MinConns: cfg.DBMinConns,
			Interval: time.Duration(cfg.PgFailoverPollMs) * time.Millisecond,
		}, func(newPool *pgxpool.Pool) {
			old := pool
			pool = newPool
			registry.SetPool(newPool)
			slotMapWatcher.SetPool(newPool)
			gnetHandler.SetPool(newPool)
			settingsWatcher.SetPGFallback(ingestion.SettingsPGSync(newPool), registry.IsStaleMode)
			if old != nil && old != newPool {
				old.Close()
			}
			slog.Info("tracker pg failover reconnected read pool")
		})
		if ingestPgFailover != nil {
			defer ingestPgFailover.Stop()
		}
	}

	var brokerProducer *ingestion.BrokerProducer
	var fraudBrokerSink *ingestion.FraudBrokerSink
	if cfg.BrokerEnabled() && cfg.BrokerPrimaryCH() && cfg.Broker.URL != "" {
		producerCfg := ingestion.DefaultBrokerProducerConfig()
		producerCfg.BrokerAddr = cfg.Broker.URL
		producerCfg.Topic = cfg.Broker.Topic
		if cfg.Broker.TimeoutMs > 0 {
			producerCfg.Timeout = time.Duration(cfg.Broker.TimeoutMs) * time.Millisecond
		}
		var bpErr error
		brokerProducer, bpErr = ingestion.NewBrokerProducer(producerCfg)
		if bpErr != nil {
			slog.Error("broker producer init failed (CH_INGEST_SOURCE=broker)", "error", bpErr)
			os.Exit(1)
		}
		gnetHandler.SetBrokerProducer(brokerProducer)
		slog.Info("broker producer enabled", "addr", cfg.Broker.URL, "topic", producerCfg.Topic)

		brokerRedisURL := cfg.Broker.RedisURL
		if brokerRedisURL == "" && len(cfg.RedisAddrs) > 0 {
			brokerRedisURL = database.BrokerRedisURL(cfg.RedisAddrs, string(cfg.RedisPassword))
		}
		fraudTimeout := producerCfg.Timeout
		fraudBrokerSink, bpErr = ingestion.NewFraudBrokerSink(cfg.Broker.URL, brokerRedisURL, cfg.Broker.FraudTopic, fraudTimeout)
		if bpErr != nil {
			slog.Error("fraud broker sink init failed (CH_INGEST_SOURCE=broker)", "error", bpErr)
			os.Exit(1)
		}
		if fw := gnetHandler.FraudWriter(); fw != nil {
			fw.SetBrokerSink(fraudBrokerSink)
		}
		slog.Info("fraud broker sink enabled", "addr", cfg.Broker.URL, "topic", cfg.Broker.FraudTopic)
	} else if cfg.Broker.CHIngestSource != "broker" && cfg.RedisStreamName != "" {
		streamProducers := make([]*ingestion.StreamProducer, len(rdbs))
		writeTimeout := time.Duration(cfg.WriteTimeoutMs) * time.Millisecond
		if writeTimeout <= 0 {
			writeTimeout = 2 * time.Second
		}
		for i, rdb := range rdbs {
			if rdb == nil {
				continue
			}
			streamProducers[i] = ingestion.NewStreamProducer(rdb, cfg.RedisStreamName, cfg.StreamMaxLen, writeTimeout)
		}
		gnetHandler.SetStreamProducers(streamProducers)
		slog.Info("redis stream producers enabled", "stream", cfg.RedisStreamName, "shards", len(rdbs))
	}
	ingestion.StartFraudBackpressureWatcher(ctx, ingestion.FraudBackpressureConfig{
		Rdbs:        rdbs,
		Writer:      gnetHandler.FraudWriter(),
		Stream:      cfg.FraudStreamName,
		EventStream: cfg.RedisStreamName,
		Group:       cfg.RedisGroupName,
		LagSec:      cfg.FraudConsumerLagSec,
		Interval:    2 * time.Second,
	})
	if udpCtrl := ingestion.NewUDPControlFromConfig(cfg, len(rdbs)); udpCtrl != nil {
		if err := udpCtrl.Start(ctx); err != nil {
			slog.Error("udp control start failed", "error", err)
			os.Exit(1)
		}
		defer func() { _ = udpCtrl.Close() }()
		gnetHandler.SetUDPControl(udpCtrl)
		slog.Info("udp ingress control enabled", "fail_closed", cfg.UDPFailClosed)
	}
	if cfg.TCPControlEnabled {
		tcpClient := ingestion.NewTCPControlClient(ingestion.TCPControlClientConfig{
			Enabled:     true,
			Secret:      []byte(cfg.TCPControlHMACSecret),
			TrackerID:   cfg.UDPTrackerID,
			ControlAddr: cfg.TCPControlAddr,
			Sharder:     sharder,
		})
		go func() {
			ticker := time.NewTicker(time.Duration(cfg.SlotMapPollIntervalMs) * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					if err := tcpClient.RequestSnapshot(ctx); err != nil {
						slog.Debug("tcp routing snapshot pull", "error", err)
					}
				}
			}
		}()
		slog.Info("tcp routing snapshot client enabled", "control_addr", cfg.TCPControlAddr)
	}
	gnetHandler.ConfigureIngestGeo(geoProvider)
	if cfg.CIDRL1Enabled {
		cidrTable := ingestion.NewCIDRTable()
		gnetHandler.ConfigureCIDR(cidrTable)
		if cidrLoader := ingestion.NewCIDRFeedLoader(cfg, cidrTable); cidrLoader != nil {
			go cidrLoader.Start(ctx)
			slog.Info("cidr l1 loader started", "dir", cfg.CIDRFeedDir, "refresh", cfg.CIDRFeedRefresh, "download", cfg.CIDRFeedDownloadEnable)
		}
	}
	if cfg.IPv6RotationL1Enabled {
		rotTable := ingestion.NewIPv6RotationTable()
		rotTable.SetMode(cfg.IPv6RotationMode)
		rotTable.SetPolicy(uint64(cfg.IPv6RotationWindow.Nanoseconds()), cfg.IPv6RotationThreshold)
		gnetHandler.ConfigureIPv6Rotation(rotTable)
		slog.Info("ipv6 rotation l1 enabled", "mode", cfg.IPv6RotationMode, "window", cfg.IPv6RotationWindow, "threshold", cfg.IPv6RotationThreshold)
	}
	if cfg.IPv4RotationL1Enabled {
		v4Rot := ingestion.NewIPv4RotationTable()
		v4Rot.SetMode(cfg.IPv4RotationMode)
		v4Rot.SetPolicy(uint64(cfg.IPv4RotationWindow.Nanoseconds()), cfg.IPv4RotationThreshold)
		gnetHandler.ConfigureIPv4Rotation(v4Rot)
		slog.Info("ipv4 rotation l1 enabled", "mode", cfg.IPv4RotationMode, "window", cfg.IPv4RotationWindow, "threshold", cfg.IPv4RotationThreshold)
	}
	if cfg.ProxyVPNL15Enabled {
		proxyTable := ingestion.NewProxyVPNTable()
		gnetHandler.ConfigureProxyVPN(proxyTable)
		if proxyLoader := ingestion.NewProxyVPNFeedLoader(cfg, proxyTable); proxyLoader != nil {
			go proxyLoader.Start(ctx)
			slog.Info("proxy vpn l1.5 loader started", "dir", cfg.ProxyVPNFeedDir, "refresh", cfg.ProxyVPNFeedRefresh)
		}
	}
	if cfg.TLSFingerprintL1Enabled {
		tlsTable := ingestion.NewTLSFingerprintTable()
		gnetHandler.ConfigureTLSFingerprint(tlsTable)
		if tlsLoader := ingestion.NewTLSFingerprintFeedLoader(cfg, tlsTable); tlsLoader != nil {
			go tlsLoader.Start(ctx)
			slog.Info("tls fingerprint l1 loader started", "dir", cfg.TLSFingerprintFeedDir, "refresh", cfg.TLSFingerprintFeedRefresh)
		}
	}
	if secret := string(cfg.LinkSigningHMACSecret); secret != "" {
		gnetHandler.ConfigureLinkSigning([]byte(secret))
		slog.Info("link signing enabled")
	}
	var attestationSecrets [][]byte
	if secret := string(cfg.AttestationHMACSecret); secret != "" {
		attestationSecrets = append(attestationSecrets, []byte(secret))
	}
	if prev := string(cfg.AttestationHMACSecretPrev); prev != "" {
		attestationSecrets = append(attestationSecrets, []byte(prev))
	}
	if len(attestationSecrets) > 0 {
		gnetHandler.ConfigureAttestation(attestationSecrets)
		slog.Info("l2 attestation cookie verify enabled")
	}
	if cfg.DomainPoolEnabled {
		domainPoolTable := ingestion.NewDomainPoolTable()
		gnetHandler.ConfigureDomainPool(domainPoolTable)
		if domainPoolSync := ingestion.NewDomainPoolSync(pool, domainPoolTable, cfg.DomainPoolSyncInterval); domainPoolSync != nil {
			go domainPoolSync.Start(ctx)
			slog.Info("domain pool sync started", "interval", cfg.DomainPoolSyncInterval)
		}
	}
	if cfg.FlowRoutingEnabled {
		flowTable := ingestion.NewCampaignFlowTable()
		gnetHandler.ConfigureCampaignFlow(flowTable)
		if flowSync := ingestion.NewCampaignFlowSync(pool, flowTable, cfg.FlowSyncInterval); flowSync != nil {
			go flowSync.Start(ctx)
			slog.Info("campaign flow sync started", "interval", cfg.FlowSyncInterval)
		}
	}
	if rtbCatalog != nil {
		gnetHandler.ConfigureRtb(rtbCatalog, geoProvider, unifiedFilter, settingsWatcher)
	}
	gnetHandler.SetLogger(appLogger)
	gnetHandler.StartHealthProbe(ctx)

	workerPool := ingestion.NewPinnedWorkerPool(cfg.MaxWorkers, 8192)
	gnetHandler.SetWorkerPool(workerPool)

	slog.Info("starting ad-event-tracker via gnet", "port", cfg.ServerPort, "unix_socket", cfg.TrackerUnixSocket)

	listenURI := "tcp://:" + cfg.ServerPort
	if cfg.TrackerUnixSocket != "" {
		if err := netaddr.PrepareUnixSocket(cfg.TrackerUnixSocket); err != nil {
			slog.Error("tracker unix socket prepare failed", "path", cfg.TrackerUnixSocket, "error", err)
			os.Exit(1)
		}
		listenURI = netaddr.GnetListenURI(cfg.TrackerUnixSocket)
	}

	go func() {
		err := gnet.Run(gnetHandler, listenURI,
			gnet.WithMulticore(true),
			gnet.WithReusePort(true),
			gnet.WithTCPNoDelay(gnet.TCPNoDelay),
			gnet.WithNumEventLoop(2),
			gnet.WithLockOSThread(false),
		)
		if err != nil {
			slog.Error("gnet server failed", "error", err)
			os.Exit(1)
		}
	}()

	metricsMux := http.NewServeMux()
	live := &lifecycle.Liveness{}
	ready := &lifecycle.ReadinessProbe{}
	ready.StartBackground(ctx, 2*time.Second, func(probeCtx context.Context) bool {
		return gnetHandler.WarmReady()
	})
	lifecycle.Register(metricsMux, live, ready)
	metricsMux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		lifecycle.ServeHealthz(live, w, r)
	})
	metricsMux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		if gnetHandler.WarmReady() {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(lifecycle.BodyOK()))
			return
		}
		http.Error(w, "not ready", http.StatusServiceUnavailable)
	})
	metricsMux.Handle("/metrics", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gnetHandler.FlushLatency()
		promhttp.Handler().ServeHTTP(w, r)
	}))
	metricsSrv := &http.Server{
		Addr:              ":" + cfg.MetricsPort,
		Handler:           metricsMux,
		ReadHeaderTimeout: 2 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
	}
	go func() {
		if err := metricsSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("metrics sidecar server failed", "error", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	sig := <-stop
	slog.Info("received shutdown signal", "signal", sig.String())

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), time.Duration(cfg.Lifecycle.ShutdownTimeoutMs)*time.Millisecond)
	defer shutdownCancel()

	cancel()

	if rtbReconcile != nil {
		rtbReconcile.Close()
		reconcileWaitCtx, reconcileWaitCancel := context.WithTimeout(context.Background(), time.Duration(cfg.Lifecycle.WaitTimeoutMs)*time.Millisecond)
		if err := rtbReconcile.Wait(reconcileWaitCtx); err != nil {
			slog.Warn("rtb budget reconcile wait failed", "error", err)
		}
		reconcileWaitCancel()
	}

	if err := gnetHandler.Stop(shutdownCtx); err != nil {
		slog.Error("gnet server shutdown failed", "error", err)
	}

	metricsShutdownCtx, metricsCancel := context.WithTimeout(context.Background(), time.Duration(cfg.Lifecycle.WaitTimeoutMs)*time.Millisecond)
	if err := metricsSrv.Shutdown(metricsShutdownCtx); err != nil {
		slog.Error("metrics server shutdown failed", "error", err)
	}
	metricsCancel()

	workerPool.Shutdown()

	if localQuantaFlusher != nil {
		flushCtx, flushCancel := context.WithTimeout(context.Background(), 2*time.Second)
		n := localQuantaFlusher.FlushAll(flushCtx)
		flushCancel()
		if n > 0 {
			slog.Info("local quanta flushed on shutdown", "campaigns", n)
		}
		ingestion.SetRegistryQuantaFlushHook(nil)
	}

	if quotaRefillWorker != nil {
		quotaRefillWorker.Close()
	}
	if budgetDeltaPublisher != nil {
		budgetDeltaPublisher.Close()
	}
	if localQuantaStream != nil {
		localQuantaStream.Close()
	}
	if brokerProducer != nil {
		if err := brokerProducer.Close(); err != nil {
			slog.Warn("broker producer close error", "error", err)
		}
	}
	if streamTrimmer != nil {
		streamTrimmer.Close()
		streamTrimmer.Wait()
	}

	registryWaitCtx, registryWaitCancel := context.WithTimeout(context.Background(), time.Duration(cfg.Lifecycle.WaitTimeoutMs)*time.Millisecond)
	defer registryWaitCancel()
	if err := registry.Wait(registryWaitCtx); err != nil {
		slog.Error("registry wait failed", "error", err)
	}

	unifiedFilter.CloseFilterEvalPins()

	for i, rdb := range rdbs {
		if rdb == nil {
			continue
		}
		if err := rdb.Close(); err != nil {
			slog.Error("failed to close redis shard", "shard", i, "error", err)
		}
	}
	slog.Info("ad-event-tracker shutdown complete")
}

func firstConnectedRedis(rdbs []redis.UniversalClient) redis.UniversalClient {
	for _, rdb := range rdbs {
		if rdb != nil {
			return rdb
		}
	}
	return nil
}
