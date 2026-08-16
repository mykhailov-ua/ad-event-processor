package config

import (
	"os"
	"strings"
)

func loadControlplaneModules(cfg *Config) {
	cfg.Notifier.WorkerIntervalMs = getEnvInt("NOTIFIER_WORKER_INTERVAL_MS", 1000)
	cfg.Notifier.WorkerBatchSize = getEnvInt("NOTIFIER_WORKER_BATCH_SIZE", 10)
	cfg.Notifier.BreakerFailThreshold = getEnvInt("NOTIFIER_BREAKER_FAIL_THRESHOLD", 3)
	cfg.Notifier.BreakerSuccessThreshold = getEnvInt("NOTIFIER_BREAKER_SUCCESS_THRESHOLD", 2)
	cfg.Notifier.BreakerOpenTimeoutMs = getEnvInt("NOTIFIER_BREAKER_OPEN_TIMEOUT_MS", 30000)
	cfg.Notifier.TelegramBotToken = Secret(os.Getenv("TELEGRAM_BOT_TOKEN"))
	cfg.Notifier.TelegramChatID = os.Getenv("TELEGRAM_CHAT_ID")
	cfg.Notifier.SlackWebhookURL = Secret(os.Getenv("SLACK_WEBHOOK_URL"))
	cfg.Notifier.SMSProviderURL = os.Getenv("SMS_PROVIDER_URL")
	cfg.Notifier.SMSAPIToken = Secret(os.Getenv("SMS_API_TOKEN"))
	cfg.Notifier.SMSDefaultRecipient = os.Getenv("SMS_DEFAULT_RECIPIENT")
	cfg.Notifier.SMTPHost = os.Getenv("SMTP_HOST")
	cfg.Notifier.SMTPPort = os.Getenv("SMTP_PORT")
	cfg.Notifier.SMTPUsername = os.Getenv("SMTP_USERNAME")
	cfg.Notifier.SMTPPassword = Secret(os.Getenv("SMTP_PASSWORD"))
	cfg.Notifier.SMTPSender = os.Getenv("SMTP_SENDER")
	cfg.Notifier.RetentionSentDays = getEnvInt("NOTIFIER_RETENTION_SENT_DAYS", 30)
	cfg.Notifier.RetentionFailedDays = getEnvInt("NOTIFIER_RETENTION_FAILED_DAYS", 7)
	cfg.Notifier.RetentionIntervalHours = getEnvInt("NOTIFIER_RETENTION_INTERVAL_HOURS", 24)
	cfg.Notifier.AdminBaseURL = os.Getenv("NOTIFIER_ADMIN_BASE_URL")
	if cfg.Notifier.AdminBaseURL == "" {
		cfg.Notifier.AdminBaseURL = cfg.ManagementURL
	}
	cfg.Notifier.WorkerConcurrency = getEnvInt("NOTIFIER_WORKER_CONCURRENCY", 1)
	cfg.Notifier.DedupCooldownSec = getEnvInt("NOTIFIER_DEDUP_COOLDOWN_SEC", 300)
	cfg.Notifier.ClaimStaleSec = getEnvInt("NOTIFIER_CLAIM_STALE_SEC", 300)
	cfg.Notifier.GroupParallelism = getEnvInt("NOTIFIER_GROUP_PARALLELISM", 2)
	cfg.Notifier.RateLimitPerMinute = getEnvInt("NOTIFIER_RATE_LIMIT_PER_MINUTE", 60)
	cfg.Notifier.TelegramRateLimitPerMinute = getEnvInt("NOTIFIER_TELEGRAM_RATE_LIMIT", 20)
	cfg.Notifier.InvoiceRecipient = os.Getenv("BILLING_INVOICE_NOTIFY_RECIPIENT")
	invoiceProvider := os.Getenv("BILLING_INVOICE_NOTIFY_PROVIDER")
	switch strings.ToUpper(invoiceProvider) {
	case "SLACK":
		cfg.Notifier.InvoiceProvider = "SLACK"
	case "SMTP":
		cfg.Notifier.InvoiceProvider = "SMTP"
	default:
		cfg.Notifier.InvoiceProvider = "TELEGRAM"
	}

	cfg.Billing.InvoiceWorkerEnabled = getEnvBool("BILLING_INVOICE_WORKER_ENABLED", true)
	cfg.Billing.PaymentProvider = envOrDefault("BILLING_PAYMENT_PROVIDER", "placeholder")
	cfg.Billing.PaymentProviderKey = Secret(os.Getenv("BILLING_PAYMENT_PROVIDER_KEY"))
	if cfg.Billing.PaymentProviderKey == "" {
		cfg.Billing.PaymentProviderKey = "placeholder_dev"
	}
	cfg.Billing.ExportFetchRows = getEnvInt("BILLING_EXPORT_FETCH_ROWS", 1000)
	cfg.Billing.ExportJobTimeoutMin = getEnvInt("BILLING_EXPORT_JOB_TIMEOUT_MIN", 15)
	cfg.BillingInternalToken = Secret(os.Getenv("BILLING_INTERNAL_TOKEN"))
}

func applyControlplaneDefaults(cfg *Config) {
	if cfg.PaymentWebhookPort == "" {
		cfg.PaymentWebhookPort = "8187"
	}
	paymentHTTPBase := "http://127.0.0.1:" + cfg.PaymentWebhookPort
	if cfg.StripeCheckoutSuccessURL == "" {
		cfg.StripeCheckoutSuccessURL = paymentHTTPBase + "/ui/payment/return?status=success&session_id={CHECKOUT_SESSION_ID}"
	}
	if cfg.StripeCheckoutCancelURL == "" {
		cfg.StripeCheckoutCancelURL = paymentHTTPBase + "/ui/payment/return?status=cancelled"
	}
}

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
	cfg.Management.SmartAlertsEnabled = getEnvBool("SMART_ALERTS_ENABLED", true)
	cfg.Management.SmartAlertsIntervalMin = getEnvInt("SMART_ALERTS_INTERVAL_MIN", 15)
	if cfg.Management.SmartAlertsIntervalMin < 5 {
		cfg.Management.SmartAlertsIntervalMin = 5
	}
	if cfg.Management.SmartAlertsIntervalMin > 60 {
		cfg.Management.SmartAlertsIntervalMin = 60
	}
	cfg.AdminDomain = strings.TrimSpace(os.Getenv("ADMIN_DOMAIN"))
	cfg.Management.DomainHealthEnabled = getEnvBool("DOMAIN_HEALTH_ENABLED", true)
	cfg.Management.DomainHealthIntervalMin = getEnvInt("DOMAIN_HEALTH_INTERVAL_MIN", 5)
	if cfg.Management.DomainHealthIntervalMin < 5 {
		cfg.Management.DomainHealthIntervalMin = 5
	}
	if cfg.Management.DomainHealthIntervalMin > 60 {
		cfg.Management.DomainHealthIntervalMin = 60
	}
	cfg.Management.DomainSSLSetupEnabled = getEnvBool("DOMAIN_SSL_SETUP_ENABLED", true)
	cfg.Management.DomainSSLSetupScript = strings.TrimSpace(os.Getenv("DOMAIN_SSL_SETUP_SCRIPT"))
	cfg.Management.DomainSSLAcmeEmail = strings.TrimSpace(os.Getenv("CADDY_ACME_EMAIL"))

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
