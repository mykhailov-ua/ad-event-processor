package config

import "os"

func loadDatabaseModules(cfg *Config) {
	cfg.PostgresFailoverEnabled = getEnvBool("PG_FAILOVER_ENABLED", false)
	cfg.PostgresFailoverCoordinator = getEnvBool("PG_FAILOVER_COORDINATOR", false)
	cfg.PostgresPrimaryDSN = Secret(envOrDefault("PG_PRIMARY_DSN", string(cfg.DBDSN)))
	cfg.PostgresStandbyDSN = Secret(os.Getenv("PG_STANDBY_DSN"))
	cfg.PostgresFailoverRedisURL = os.Getenv("PG_FAILOVER_REDIS_URL")
	cfg.PostgresFailoverHealthMs = getEnvInt("PG_FAILOVER_HEALTH_MS", 750)
	cfg.PostgresFailoverPollMs = getEnvInt("PG_FAILOVER_POLL_MS", 300)
	cfg.PostgresFailoverCoordMs = getEnvInt("PG_FAILOVER_COORD_MS", 350)
	cfg.PostgresFailoverLeaseSec = getEnvInt("PG_FAILOVER_LEASE_SEC", 3)
	cfg.PostgresFailoverFailThreshold = getEnvInt("PG_FAILOVER_FAIL_THRESHOLD", 2)
	cfg.PostgresPromoteCommand = os.Getenv("PG_PROMOTE_COMMAND")
	cfg.PostgresFailoverSnapshotSync = getEnvBool("PG_FAILOVER_SNAPSHOT_SYNC", false)
	cfg.PostgresFailoverSyncPageSize = getEnvInt("PG_FAILOVER_SYNC_PAGE_SIZE", 5000)
	cfg.PostgresFailoverAuditWindowSec = getEnvInt("PG_FAILOVER_AUDIT_WINDOW_SEC", 3600)
	cfg.QuotaAutoRepair = getEnvBool("QUOTA_AUTO_REPAIR", false)
}
