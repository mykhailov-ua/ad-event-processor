package config

import "os"

func loadManagementModules(cfg *Config) {
	cfg.IVT.Enabled = getEnvBool("IVT_DETECTOR_ENABLED", true)
	cfg.IVT.ScanIntervalMs = getEnvInt("IVT_DETECTOR_SCAN_INTERVAL_MS", 300000)
	cfg.IVT.OutboxPendingLimit = getEnvInt64("IVT_DETECTOR_OUTBOX_PENDING_LIMIT", 500)
	cfg.IVT.WindowSec = getEnvInt("IVT_DETECTOR_WINDOW_SEC", 3600)
	cfg.IVT.MinClicks = uint64(getEnvInt64("IVT_DETECTOR_MIN_CLICKS", 10))
	cfg.IVT.MinImpressions = uint64(getEnvInt64("IVT_DETECTOR_MIN_IMPRESSIONS", 1))
	cfg.IVT.ClickToImpRatio = getEnvFloat("IVT_DETECTOR_CLICK_TO_IMP_RATIO", 5.0)
	cfg.IVT.MinIPsPerUA = uint64(getEnvInt64("IVT_DETECTOR_MIN_IPS_PER_UA", 8))
	cfg.IVT.IntervalMinIntervals = uint64(getEnvInt64("IVT_DETECTOR_INTERVAL_MIN_INTERVALS", 30))
	cfg.IVT.IntervalMaxVariance = getEnvFloat("IVT_DETECTOR_INTERVAL_MAX_VARIANCE", 0.005)

	cfg.FraudScoring.Enabled = getEnvBool("FRAUD_SCORING_ENABLED", false)
	cfg.FraudScoring.ScanIntervalMs = getEnvInt("FRAUD_SCORING_SCAN_INTERVAL_MS", 300000)
	cfg.FraudScoring.BatchSize = getEnvInt("FRAUD_SCORING_BATCH_SIZE", 1000)
	cfg.FraudScoring.ModelPath = os.Getenv("FRAUD_SCORING_MODEL_PATH")
	if cfg.FraudScoring.ModelPath == "" {
		cfg.FraudScoring.ModelPath = "var/fraudscore/artifacts/model.txt"
	}
	cfg.FraudScoring.Standalone = getEnvBool("FRAUD_SCORER_STANDALONE", false)

	if len(cfg.AllowedOrigins) == 1 && cfg.AllowedOrigins[0] == "" {
		cfg.AllowedOrigins = []string{"https://dashboard.example.com", "http://localhost:8188"}
	}

	cfg.Management.RetentionDays = getEnvInt("MANAGEMENT_RETENTION_DAYS", 90)
	cfg.Management.CancellationFeePercent = getEnvFloat("MANAGEMENT_CANCELLATION_FEE_PERCENT", 5.0)
	cfg.Management.ReconIntervalMs = getEnvInt("RECON_WORKER_INTERVAL_MS", 3_600_000)
	cfg.Management.ReconSnapshotIntervalMs = getEnvInt("RECON_SNAPSHOT_INTERVAL_MS", 0)
	cfg.Management.PacingIntervalMs = getEnvInt("PACING_CONTROLLER_INTERVAL_MS", 300_000)
	cfg.Management.RateLimitRPS = getEnvFloat("MANAGEMENT_RATE_LIMIT_RPS", 10)
	cfg.Management.RateLimitBurst = getEnvInt("MANAGEMENT_RATE_LIMIT_BURST", 50)
	cfg.Management.OpsAlertsEnabled = getEnvBool("OPS_ALERTS_ENABLED", false)
	cfg.Management.OpsAlertCooldownSec = getEnvInt("OPS_ALERT_COOLDOWN_SEC", 300)
	cfg.Management.DrainStuckThresholdSec = getEnvInt("OPS_ALERT_DRAIN_STUCK_SEC", 900)
	cfg.Management.BlacklistJanitorEnabled = getEnvBool("BLACKLIST_JANITOR_ENABLED", true)
	cfg.Management.BlacklistJanitorIntervalSec = getEnvInt("BLACKLIST_JANITOR_INTERVAL_SEC", 60)
	cfg.Management.BlacklistAutoTTLHours = getEnvInt("BLACKLIST_AUTO_TTL_HOURS", 24)
	cfg.Management.BlacklistFraudTTLHours = getEnvInt("BLACKLIST_FRAUD_TTL_HOURS", 168)
	cfg.Management.AlertmanagerWebhookEnabled = getEnvBool("ALERTMANAGER_WEBHOOK_ENABLED", false)
	cfg.Management.AlertmanagerWebhookToken = os.Getenv("ALERTMANAGER_WEBHOOK_TOKEN")
	cfg.Management.OpsAlertOutboxStuckSec = getEnvInt("OPS_ALERT_OUTBOX_STUCK_SEC", 120)
	cfg.Management.AuditExportPath = os.Getenv("AUDIT_EXPORT_PATH")
	if cfg.Management.AuditExportPath == "" {
		cfg.Management.AuditExportPath = "./data/audit-export"
	}
	cfg.Management.AuditExportRetentionDays = getEnvInt("AUDIT_EXPORT_RETENTION_DAYS", 90)
	cfg.Management.BillingExportPath = os.Getenv("BILLING_EXPORT_PATH")
	if cfg.Management.BillingExportPath == "" {
		cfg.Management.BillingExportPath = "./data/billing-export"
	}
	cfg.Management.SupplyExportPath = os.Getenv("SUPPLY_EXPORT_PATH")
	if cfg.Management.SupplyExportPath == "" {
		cfg.Management.SupplyExportPath = "./data/supply-export"
	}
	cfg.Management.AdminFanoutMaxConcurrency = getEnvInt("ADMIN_FANOUT_MAX_CONCURRENCY", 8)
	cfg.Management.LowBalanceThresholdMicro = int64(getEnvInt("LOW_BALANCE_THRESHOLD_MICRO", 5_000_000))
	cfg.Management.LowBalanceAlertEnabled = getEnvBool("LOW_BALANCE_ALERT_ENABLED", true)

	cfg.Control.EnableAuth = getEnvBool("CONTROL_ENABLE_AUTH", true)
	cfg.Control.EnableManagement = getEnvBool("CONTROL_ENABLE_MANAGEMENT", true)
	cfg.Control.EnablePayment = getEnvBool("CONTROL_ENABLE_PAYMENT", true)
	cfg.Control.EnableBilling = getEnvBool("CONTROL_ENABLE_BILLING", true)
	cfg.Control.EnableNotifier = getEnvBool("CONTROL_ENABLE_NOTIFIER", true)
	cfg.Control.EnableMarginGuard = getEnvBool("CONTROL_ENABLE_MARGIN_GUARD", true)
	cfg.Control.EnableCostSync = getEnvBool("CONTROL_ENABLE_COST_SYNC", true)

	cfg.GeoIP.DBPath = os.Getenv("GEOIP_DB_PATH")
	if cfg.GeoIP.DBPath == "" {
		cfg.GeoIP.DBPath = "deploy/geoip/GeoLite2-Country.mmdb"
	}
	cfg.GeoIP.StagingPath = os.Getenv("GEOIP_STAGING_PATH")
	if cfg.GeoIP.StagingPath == "" {
		cfg.GeoIP.StagingPath = cfg.GeoIP.DBPath + ".staging"
	}
	cfg.GeoIP.EditionID = os.Getenv("MAXMIND_EDITION_ID")
	if cfg.GeoIP.EditionID == "" {
		cfg.GeoIP.EditionID = "GeoLite2-Country"
	}
	cfg.GeoIP.LicenseKey = os.Getenv("MAXMIND_LICENSE_KEY")
	cfg.GeoIP.UpdaterEnabled = getEnvBool("GEOIP_UPDATER_ENABLED", false)
	cfg.GeoIP.UpdateIntervalHours = getEnvInt("GEOIP_UPDATE_INTERVAL_HOURS", 24)
	cfg.GeoIP.WatcherIntervalSec = getEnvInt("GEOIP_WATCHER_INTERVAL_SEC", 60)

	cfg.Lifecycle.ShutdownTimeoutMs = getEnvInt("SHUTDOWN_TIMEOUT_MS", 15000)
	cfg.Lifecycle.DrainTimeoutMs = getEnvInt("DRAIN_TIMEOUT_MS", 10000)
	cfg.Lifecycle.WaitTimeoutMs = getEnvInt("WAIT_TIMEOUT_MS", 5000)
}
