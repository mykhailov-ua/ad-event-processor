package notify

import (
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/notify/db"
)

type Config struct {
	TelegramBotToken    string
	TelegramChatID      string
	SlackWebhookURL     string
	SMTPHost            string
	SMTPPort            string
	SMTPUsername        string
	SMTPPassword        string
	SMTPSender          string
	SMSProviderURL      string
	SMSAPIToken         string
	SMSDefaultRecipient string
	RequireCredentials  bool
	FailTelegram        bool
	FailSlack           bool
	FailSMS             bool
	FailSMTP            bool
}

type Breakers struct {
	Telegram *CircuitBreaker
	Slack    *CircuitBreaker
	SMTP     *CircuitBreaker
	SMS      *CircuitBreaker
}

func ConfigFromApp(cfg *config.Config) Config {
	if cfg == nil {
		return Config{}
	}
	n := cfg.Notifier
	return Config{
		TelegramBotToken:    string(n.TelegramBotToken),
		TelegramChatID:      n.TelegramChatID,
		SlackWebhookURL:     string(n.SlackWebhookURL),
		SMTPHost:            n.SMTPHost,
		SMTPPort:            n.SMTPPort,
		SMTPUsername:        n.SMTPUsername,
		SMTPPassword:        string(n.SMTPPassword),
		SMTPSender:          n.SMTPSender,
		SMSProviderURL:      n.SMSProviderURL,
		SMSAPIToken:         string(n.SMSAPIToken),
		SMSDefaultRecipient: n.SMSDefaultRecipient,
		RequireCredentials:  isProdEnv(cfg.Env),
	}
}

func BreakersFromApp(cfg *config.Config) Breakers {
	if cfg == nil {
		return Breakers{}
	}
	n := cfg.Notifier
	return NewBreakers(int64(n.BreakerFailThreshold), int64(n.BreakerSuccessThreshold), time.Duration(n.BreakerOpenTimeoutMs)*time.Millisecond)
}

func NewBreakers(failThreshold, successThreshold int64, openTimeout time.Duration) Breakers {
	return Breakers{
		Telegram: NewCircuitBreaker(failThreshold, successThreshold, openTimeout),
		Slack:    NewCircuitBreaker(failThreshold, successThreshold, openTimeout),
		SMTP:     NewCircuitBreaker(failThreshold, successThreshold, openTimeout),
		SMS:      NewCircuitBreaker(failThreshold, successThreshold, openTimeout),
	}
}

func isProdEnv(env string) bool {
	return env == "production" || env == "prod"
}

func (c Config) providerConfigured(p db.NotifierProvider) bool {
	if !c.RequireCredentials {
		return true
	}
	switch p {
	case db.NotifierProviderTELEGRAM:
		return c.TelegramBotToken != ""
	case db.NotifierProviderSLACK:
		return c.SlackWebhookURL != ""
	case db.NotifierProviderSMTP:
		return c.SMTPHost != "" && c.SMTPSender != ""
	case db.NotifierProviderSMS:
		return c.SMSProviderURL != ""
	default:
		return false
	}
}
