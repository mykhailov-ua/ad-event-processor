package config

import (
	"fmt"
	"os"
	"time"
)

func loadIngestModules(cfg *Config, appEnv string) error {
	cfg.Logger.Dir = os.Getenv("LOGGER_DIR")
	if cfg.Logger.Dir == "" {
		cfg.Logger.Dir = "/var/log/espx"
	}
	cfg.Logger.Shards = getEnvInt("LOGGER_SHARDS", 8)
	cfg.Logger.FlushSizeKB = getEnvInt("LOGGER_FLUSH_SIZE_KB", 256)
	cfg.Logger.RotateSizeMB = getEnvInt("LOGGER_ROTATE_SIZE_MB", 512)
	cfg.Logger.RotateInterval = time.Duration(getEnvInt("LOGGER_ROTATE_INTERVAL_MIN", 60)) * time.Minute
	cfg.Logger.LatencyLimit = time.Duration(getEnvInt("LOGGER_LATENCY_LIMIT_MS", 100)) * time.Millisecond
	cfg.Logger.PersistQueueDepth = getEnvInt("LOGGER_PERSIST_QUEUE_DEPTH", 0)
	cfg.Logger.PersistEnqueueTimeout = time.Duration(getEnvInt("LOGGER_PERSIST_ENQUEUE_TIMEOUT_MS", 25)) * time.Millisecond

	cfg.Broker.URL = os.Getenv("BROKER_URL")
	cfg.Broker.RedisURL = os.Getenv("BROKER_REDIS_URL")
	cfg.Broker.Topic = os.Getenv("BROKER_TOPIC")
	cfg.Broker.PartitionCount = getEnvInt("BROKER_PARTITION_COUNT", ExpectedRedisShardCount)
	cfg.Broker.ShadowMode = getEnvBool("BROKER_SHADOW_MODE", true)
	cfg.Broker.MaxBytes = getEnvInt("BROKER_FETCH_MAX_BYTES", 1024*1024)
	cfg.Broker.TimeoutMs = getEnvInt("BROKER_TIMEOUT_MS", 5000)
	cfg.Broker.ReconcileIntervalMs = getEnvInt("BROKER_RECONCILE_INTERVAL_MS", 30000)
	cfg.Broker.DivergenceThreshold = uint64(getEnvInt64("BROKER_DIVERGENCE_THRESHOLD", 1000))
	if cfg.Broker.Topic == "" {
		cfg.Broker.Topic = "tracker-logs"
	}

	cfg.RtbMode = os.Getenv("RTB_MODE")
	cfg.RtbBudgetAuthority = os.Getenv("RTB_BUDGET_AUTHORITY")
	cfg.RtbClearingMode = os.Getenv("RTB_CLEARING_MODE")
	cfg.RtbSnapshotPath = os.Getenv("RTB_SNAPSHOT_PATH")
	cfg.RtbHybridMaxRpsPerNode = getEnvInt("RTB_HYBRID_MAX_RPS_PER_NODE", 0)
	cfg.RtbReconcileIntervalMs = getEnvInt("RTB_RECONCILE_INTERVAL_MS", 30000)
	cfg.RtbBudgetDivergenceMicro = int64(getEnvInt("RTB_BUDGET_DIVERGENCE_THRESHOLD_MICRO", 1000))
	cfg.RtbReconcileSampleSize = getEnvInt("RTB_RECONCILE_SAMPLE_SIZE", 32)
	cfg.RtbTargetingIndex = getEnvBool("RTB_TARGETING_INDEX", true)
	cfg.RtbPrebidIVT = getEnvBool("RTB_PREBID_IVT", false)
	if cfg.RtbBudgetAuthority == "" {
		cfg.RtbBudgetAuthority = "redis"
	}

	rawIngress := os.Getenv("TRACKER_INGRESS_SCHEMA")
	if rawIngress == "" {
		rawIngress = IngressSchemaOpenRTB3
	}
	cfg.IngressSchema = NormalizeIngressSchema(rawIngress)
	switch cfg.IngressSchema {
	case IngressSchemaOpenRTB3, IngressSchemaNativeV1:
	default:
		return fmt.Errorf("invalid TRACKER_INGRESS_SCHEMA %q (want openrtb_3 or native_v1)", cfg.IngressSchema)
	}

	cfg.QuotaMode = os.Getenv("QUOTA_MODE")
	if cfg.QuotaMode == "" {
		cfg.QuotaMode = "off"
	}
	cfg.LocalQuotaMode = os.Getenv("LOCAL_QUOTA_MODE")
	if cfg.LocalQuotaMode == "" {
		cfg.LocalQuotaMode = "off"
	}
	cfg.QuotaChunkSize = getEnvInt64("QUOTA_CHUNK_SIZE", 0)
	cfg.QuotaStrictThresholdMicro = getEnvInt64("QUOTA_STRICT_THRESHOLD_MICRO", 5_000_000)
	cfg.QuotaStrictExitMicro = getEnvInt64("QUOTA_STRICT_EXIT_MICRO", 8_000_000)
	cfg.QuotaRefillThresholdPct = getEnvInt("QUOTA_REFILL_THRESHOLD_PCT", 20)
	cfg.LocalQuotaRefillMaxShard = getEnvInt("LOCAL_QUOTA_REFILL_MAX_PER_SHARD", 4)
	cfg.QuotaAdaptiveFloorMicro = getEnvInt64("QUOTA_ADAPTIVE_FLOOR_MICRO", 500_000)
	cfg.QuotaAdaptiveCeilingMicro = getEnvInt64("QUOTA_ADAPTIVE_CEILING_MICRO", 50_000_000)
	cfg.BudgetDeltaTopic = os.Getenv("BUDGET_DELTA_TOPIC")
	if cfg.BudgetDeltaTopic == "" {
		cfg.BudgetDeltaTopic = "budget-deltas"
	}

	cfg.SlotMapReloadTopic = os.Getenv("SLOT_MAP_RELOAD_TOPIC")
	if cfg.SlotMapReloadTopic == "" {
		cfg.SlotMapReloadTopic = "shards:reload"
	}
	cfg.SlotMapPollIntervalMs = getEnvInt("SLOT_MAP_POLL_INTERVAL_MS", 10000)
	cfg.SlotMigrationEnabled = getEnvBool("SLOT_MIGRATION_ENABLED", true)
	cfg.SlotMigrationIntervalMs = getEnvInt("SLOT_MIGRATION_INTERVAL_MS", 30000)
	cfg.MigrationFenceEnabled = getEnvBool("MIGRATION_FENCE_ENABLED", appEnv == "production")
	cfg.SlotMigrationDualWriteEnabled = getEnvBool("SLOT_MIGRATION_DUAL_WRITE_ENABLED", false)
	cfg.SlotMigrationLagEpsilon = getEnvInt64("SLOT_MIGRATION_LAG_EPSILON", 0)
	cfg.SlotMigrationLagThreshold = getEnvInt64("SLOT_MIGRATION_LAG_THRESHOLD", 1000)
	return nil
}
