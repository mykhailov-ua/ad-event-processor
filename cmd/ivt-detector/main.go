package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/database"
	"github.com/bidshard/ad-event-processor/internal/edge"
	"github.com/bidshard/ad-event-processor/internal/fraud"
	"github.com/bidshard/ad-event-processor/internal/licensing"
	"github.com/bidshard/ad-event-processor/pkg/lifecycle"
	"github.com/bidshard/ad-event-processor/pkg/netaddr"
	"github.com/bidshard/ad-event-processor/pkg/piihash"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/redis/go-redis/v9"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}
	if !cfg.IVTDetectorEnabled() {
		slog.Error("ivt detector requires IVT_DETECTOR_ENABLED=true and CH_DSN")
		os.Exit(1)
	}

	piiHasher, piiErr := piihash.NewFromConfig(cfg)
	if piiErr != nil {
		slog.Error("failed to initialize PII hasher", "error", piiErr)
		os.Exit(1)
	}
	fraud.SetPIIHasher(piiHasher)

	ctx, stop := lifecycle.NotifyContext(context.Background())
	defer stop()

	pool, err := database.Connect(ctx, string(cfg.DBDSN), cfg.DBTrackerMaxConns, cfg.DBMinConns)
	if err != nil {
		slog.Error("failed to connect to postgres", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	chRead, err := database.ConnectCHReadonly(ctx, string(cfg.CHReadonlyDSN))
	if err != nil {
		slog.Error("failed to connect to clickhouse readonly", "error", err)
		os.Exit(1)
	}
	defer func() { _ = chRead.Close() }()

	chQuery := database.NewCHQuery(chRead, database.CHQueryConfigFromApp(cfg))

	analyzerCfg := fraud.AnalyzerConfig{
		Window:               time.Duration(cfg.IVT.WindowSec) * time.Second,
		MinClicks:            cfg.IVT.MinClicks,
		MinImpressions:       cfg.IVT.MinImpressions,
		ClickToImpRatio:      cfg.IVT.ClickToImpRatio,
		MinIPsPerUA:          cfg.IVT.MinIPsPerUA,
		MinEventsPerIP:       cfg.IVT.MinClicks,
		IntervalMinIntervals: cfg.IVT.IntervalMinIntervals,
		IntervalMaxVariance:  cfg.IVT.IntervalMaxVariance,
	}
	detectorCfg := fraud.DetectorConfig{
		ScanInterval:       time.Duration(cfg.IVT.ScanIntervalMs) * time.Millisecond,
		OutboxPendingLimit: cfg.IVT.OutboxPendingLimit,
		Analyzer:           analyzerCfg,
	}

	var blocker fraud.BlacklistBlocker
	blocker, err = fraud.ResolveManagementBlockerFromConfig(cfg.ManagementURL, cfg.ManagementPort, string(cfg.AdminAPIKey))
	if err != nil {
		slog.Error("failed to configure management client", "error", err)
		os.Exit(1)
	}
	slog.Info("ivt detector using management HTTP API")

	asn := &fraud.StaticASNClassifier{
		DatacenterPrefixes: strings.Split(os.Getenv("IVT_DATACENTER_PREFIXES"), ","),
	}

	var scorer fraud.Scorer
	if cfg.FraudScoringEnabled() && !cfg.FraudScorerStandalone() {
		var err error
		scorer, err = fraud.NewLGBMScorer(cfg.FraudScoring.ModelPath)
		if err != nil {
			slog.Error("failed to initialize fraud scorer", "error", err, "path", cfg.FraudScoring.ModelPath)
			os.Exit(1)
		}
		slog.Info("initialized embedded fraud scorer", "path", cfg.FraudScoring.ModelPath)
	} else if cfg.FraudScorerStandalone() {
		slog.Info("FRAUD_SCORER_STANDALONE enabled; skipping embedded scorer in ivt-detector")
	}

	var chWrite driver.Conn
	if scorer != nil && string(cfg.CHDSN) != "" {
		chWrite, err = database.ConnectClickHouse(ctx, string(cfg.CHDSN))
		if err != nil {
			slog.Error("failed to connect to clickhouse write path for shadow scores", "error", err)
			os.Exit(1)
		}
		defer func() { _ = chWrite.Close() }()
	}

	snap, err := licensing.LoadDeploymentSnapshot(ctx, pool)
	if err != nil {
		slog.Warn("ivt detector: license_status unavailable; continuing with deployment defaults", "error", err)
	} else if !snap.ModuleAllowed(func(f licensing.FeatureSet) bool { return f.IvtMLEnabled() }) {
		slog.Error("ivt_ml_detector module not licensed; exiting")
		os.Exit(1)
	}

	rdb := newRedisShard0(cfg)

	registry := fraud.NewAnalyzerRegistry(chQuery, chWrite, pool, analyzerCfg, asn, scorer, cfg.FraudScoring.BatchSize, rdb)

	detector := fraud.NewDetector(
		registry,
		fraud.NewIdempotencyStore(pool),
		blocker,
		pool,
		detectorCfg,
	)

	slog.Info("starting ivt detector",
		"scan_interval_ms", cfg.IVT.ScanIntervalMs,
		"window_sec", cfg.IVT.WindowSec,
	)

	// Shutdown order when RunLoop exits: cancel via ctx → close Redis → PG/CH (defer LIFO).
	if rdb != nil {
		defer func() { _ = rdb.Close() }()
	}

	if err := detector.RunLoop(ctx); err != nil && !errors.Is(err, context.Canceled) {
		slog.Error("ivt detector stopped with error", "error", err)
		os.Exit(1)
	}

	slog.Info("ivt detector shutdown complete")
}

func newRedisShard0(cfg *config.Config) *redis.Client {
	addr := edge.FirstRedisAddr()
	if addr == "" {
		return nil
	}
	return redis.NewClient(netaddr.RedisClientOptions(addr, string(cfg.RedisPassword)))
}
