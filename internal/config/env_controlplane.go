package config

import (
	"os"
	"strings"
)

func loadControlplaneModules(cfg *Config) {
	cfg.Notifier.ServerHost = os.Getenv("NOTIFIER_SERVER_HOST")
	if cfg.Notifier.ServerHost == "" {
		cfg.Notifier.ServerHost = "127.0.0.1"
	}
	cfg.Notifier.Port = os.Getenv("NOTIFIER_PORT")
	if cfg.Notifier.Port == "" {
		cfg.Notifier.Port = "8085"
	}
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
	cfg.Notifier.MetricsPort = os.Getenv("NOTIFIER_METRICS_PORT")
	if cfg.Notifier.MetricsPort == "" {
		cfg.Notifier.MetricsPort = "8086"
	}
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

	cfg.Billing.Port = os.Getenv("BILLING_SERVER_PORT")
	if cfg.Billing.Port == "" {
		cfg.Billing.Port = "51054"
	}
	cfg.Billing.ServerHost = os.Getenv("BILLING_SERVER_HOST")
	if cfg.Billing.ServerHost == "" {
		cfg.Billing.ServerHost = "127.0.0.1"
	}
	cfg.Billing.MetricsPort = os.Getenv("BILLING_METRICS_PORT")
	if cfg.Billing.MetricsPort == "" {
		cfg.Billing.MetricsPort = "9092"
	}
	cfg.Billing.InvoiceWorkerEnabled = getEnvBool("BILLING_INVOICE_WORKER_ENABLED", true)
	cfg.Billing.PaymentProvider = envOrDefault("BILLING_PAYMENT_PROVIDER", "placeholder")
	cfg.Billing.PaymentProviderKey = Secret(os.Getenv("BILLING_PAYMENT_PROVIDER_KEY"))
	if cfg.Billing.PaymentProviderKey == "" {
		cfg.Billing.PaymentProviderKey = "placeholder_dev"
	}
	cfg.BillingInternalToken = Secret(os.Getenv("BILLING_INTERNAL_TOKEN"))
}

func applyControlplaneDefaults(cfg *Config) {
	if cfg.PaymentServerPort == "" {
		cfg.PaymentServerPort = "51052"
	}
	if cfg.PaymentServerHost == "" {
		cfg.PaymentServerHost = "127.0.0.1"
	}
	if cfg.PaymentMetricsPort == "" {
		cfg.PaymentMetricsPort = "9092"
	}
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
	if cfg.SettlementServerPort == "" {
		cfg.SettlementServerPort = "51053"
	}
	if cfg.SettlementServerHost == "" {
		cfg.SettlementServerHost = "127.0.0.1"
	}
}
