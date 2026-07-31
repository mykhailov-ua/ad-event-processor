package config

import "os"

func loadDatabaseModules(cfg *Config) {
	cfg.PgFailoverEnabled = getEnvBool("PG_FAILOVER_ENABLED", false)
	cfg.PgFailoverCoordinator = getEnvBool("PG_FAILOVER_COORDINATOR", false)
	cfg.PgPrimaryDSN = Secret(envOrDefault("PG_PRIMARY_DSN", string(cfg.DBDSN)))
	cfg.PgStandbyDSN = Secret(os.Getenv("PG_STANDBY_DSN"))
	cfg.PgFailoverRedisURL = os.Getenv("PG_FAILOVER_REDIS_URL")
	if cfg.PgFailoverRedisURL == "" && len(cfg.RedisAddrs) > 0 {
		cfg.PgFailoverRedisURL = "redis://" + cfg.RedisAddrs[0] + "/0"
	}
	cfg.PgFailoverHealthMs = getEnvInt("PG_FAILOVER_HEALTH_MS", 750)
	cfg.PgFailoverPollMs = getEnvInt("PG_FAILOVER_POLL_MS", 300)
	cfg.PgFailoverCoordMs = getEnvInt("PG_FAILOVER_COORD_MS", 350)
	cfg.PgFailoverLeaseSec = getEnvInt("PG_FAILOVER_LEASE_SEC", 3)
	cfg.PgFailoverFailThreshold = getEnvInt("PG_FAILOVER_FAIL_THRESHOLD", 2)
	cfg.PgPromoteCommand = os.Getenv("PG_PROMOTE_COMMAND")
	cfg.PgFailoverSnapshotSync = getEnvBool("PG_FAILOVER_SNAPSHOT_SYNC", false)
	cfg.PgFailoverSyncPageSize = getEnvInt("PG_FAILOVER_SYNC_PAGE_SIZE", 5000)
	cfg.PgFailoverAuditWindowSec = getEnvInt("PG_FAILOVER_AUDIT_WINDOW_SEC", 3600)
	cfg.QuotaAutoRepair = getEnvBool("QUOTA_AUTO_REPAIR", false)
}
