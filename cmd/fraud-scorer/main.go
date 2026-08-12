package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/database"
	"github.com/bidshard/ad-event-processor/internal/fraud"
	"github.com/bidshard/ad-event-processor/pkg/lifecycle"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	if !cfg.FraudScoringEnabled() {
		slog.Error("fraud-scorer requires FRAUD_SCORING_ENABLED=true")
		os.Exit(1)
	}

	artifactDir := os.Getenv("FRAUD_ARTIFACT_DIR")
	if artifactDir == "" {
		artifactDir = "var/fraudscore/artifacts"
	}
	metadataPath := filepath.Join(artifactDir, "metadata.json")
	policySource := os.Getenv("FRAUD_POLICY_SOURCE")
	if policySource == "" {
		policySource = "auto"
	}
	policyCfg := fraud.ResolvePolicyConfig(
		fraud.PolicyConfigFromEnv(),
		metadataPath,
		policySource,
	)
	fraud.SetPolicyConfig(policyCfg)
	slog.Info("fraud policy loaded",
		"source", policySource,
		"ml_threshold", policyCfg.MLThreshold,
		"proxy_floor", policyCfg.ResidentialProxyFloor,
		"fp_guard_cap", policyCfg.FPGuardCap,
	)

	ctx, stop := lifecycle.NotifyContext(context.Background())
	defer stop()

	pool, err := database.Connect(ctx, string(cfg.DBDSN), cfg.DBTrackerMaxConns, cfg.DBMinConns)
	if err != nil {
		slog.Error("failed to connect to postgres", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	chConn, err := database.ConnectClickHouse(ctx, string(cfg.CHDSN))
	if err != nil {
		slog.Error("failed to connect to clickhouse", "error", err)
		os.Exit(1)
	}
	defer func() { _ = chConn.Close() }()

	go watchAndRegisterModels(ctx, pool)

	var scorer fraud.Scorer
	scorer, err = fraud.NewLGBMScorer(cfg.FraudScoring.ModelPath)
	if err != nil {
		slog.Error("failed to initialize fraud scorer", "error", err, "path", cfg.FraudScoring.ModelPath)
		os.Exit(1)
	}
	slog.Info("initialized fraud scorer", "path", cfg.FraudScoring.ModelPath)

	analyzerCfg := fraud.AnalyzerConfig{
		Window:          time.Duration(cfg.IVT.WindowSec) * time.Second,
		MinClicks:       cfg.IVT.MinClicks,
		MinImpressions:  cfg.IVT.MinImpressions,
		ClickToImpRatio: cfg.IVT.ClickToImpRatio,
		MinIPsPerUA:     cfg.IVT.MinIPsPerUA,
		MinEventsPerIP:  cfg.IVT.MinClicks,
	}
	detectorCfg := fraud.DetectorConfig{
		ScanInterval:       time.Duration(cfg.FraudScoring.ScanIntervalMs) * time.Millisecond,
		OutboxPendingLimit: cfg.IVT.OutboxPendingLimit,
		Analyzer:           analyzerCfg,
	}

	var blocker fraud.BlacklistBlocker
	blocker, err = fraud.ResolveManagementBlockerFromConfig(cfg.ManagementURL, cfg.ManagementPort, string(cfg.AdminAPIKey))
	if err != nil {
		slog.Error("failed to configure management client", "error", err)
		os.Exit(1)
	}
	slog.Info("fraud-scorer using management HTTP API")

	registry := fraud.NewRuleRegistry()
	chQuery := database.NewCHQuery(chConn, database.CHQueryConfigFromApp(cfg))
	registry.Register(fraud.NewFraudScoringRule(chQuery, chConn, pool, scorer, cfg.FraudScoring.BatchSize))

	detector := fraud.NewDetector(
		registry,
		fraud.NewIdempotencyStore(pool),
		blocker,
		pool,
		detectorCfg,
	)

	slog.Info("starting fraud-scorer worker",
		"scan_interval_ms", cfg.FraudScoring.ScanIntervalMs,
		"window_sec", cfg.IVT.WindowSec,
	)

	if err := detector.RunLoop(ctx); err != nil && err != context.Canceled {
		slog.Error("fraud-scorer worker stopped with error", "error", err)
		os.Exit(1)
	}
}

func watchAndRegisterModels(ctx context.Context, pool *pgxpool.Pool) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := scanAndRegister(ctx, pool); err != nil {
				slog.Error("failed to scan and register models", "error", err)
			}
		}
	}
}

func scanAndRegister(ctx context.Context, pool *pgxpool.Pool) error {
	artifactDir := "var/fraudscore/artifacts"
	modelPath := filepath.Join(artifactDir, "model.txt")
	metadataPath := filepath.Join(artifactDir, "metadata.json")

	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		return nil
	}

	hash, err := calculateSHA256(modelPath)
	if err != nil {
		return fmt.Errorf("calculate sha256 of model: %w", err)
	}

	var exists bool
	err = pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM ml_model_versions WHERE artifact_hash = $1)", hash).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check model version existence: %w", err)
	}
	if exists {
		return nil
	}

	version := "v" + hash[:8]
	metricsJSON := []byte("{}")

	if _, err := os.Stat(metadataPath); err == nil {
		data, err := os.ReadFile(metadataPath)
		if err == nil {
			var meta struct {
				Version string          `json:"version"`
				Metrics json.RawMessage `json:"metrics"`
			}
			if err := json.Unmarshal(data, &meta); err == nil {
				metricsJSON = data
				if meta.Version != "" {
					version = meta.Version
				}
			}
		}
	}

	var syncingExists bool
	err = pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM ml_model_versions WHERE status = 'SYNCING')").Scan(&syncingExists)
	if err != nil {
		return fmt.Errorf("check syncing model version: %w", err)
	}
	if syncingExists {
		slog.Info("new model found but another sync is in progress; skipping registration", "version", version)
		return nil
	}

	slog.Info("registering newly trained model version", "version", version, "hash", hash)
	_, err = pool.Exec(ctx, `
		INSERT INTO ml_model_versions (id, artifact_hash, metrics_json, status, created_at)
		VALUES ($1, $2, $3, 'SYNCING', NOW())
		ON CONFLICT (id) DO NOTHING`,
		version, hash, metricsJSON)
	if err != nil {
		return fmt.Errorf("insert model version: %w", err)
	}

	return nil
}

func calculateSHA256(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}
