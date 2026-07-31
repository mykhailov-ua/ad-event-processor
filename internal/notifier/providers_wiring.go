package notifier

import (
	"context"
	"time"

	"espx/internal/config"
	"espx/internal/notifier/db"
)

type ProviderBundle struct {
	Providers map[db.NotifierProvider]Provider
	Breakers  map[db.NotifierProvider]*CircuitBreaker
}

func isProdEnv(env string) bool {
	return env == "production" || env == "prod"
}

func NewProvidersFromConfig(cfg *config.Config) map[db.NotifierProvider]Provider {
	return NewProviderBundleFromConfig(cfg).Providers
}

func NewProviderBundleFromConfig(cfg *config.Config) ProviderBundle {
	if cfg == nil {
		return ProviderBundle{}
	}

	n := cfg.Notifier
	failThreshold := int64(n.BreakerFailThreshold)
	successThreshold := int64(n.BreakerSuccessThreshold)
	openTimeout := time.Duration(n.BreakerOpenTimeoutMs) * time.Millisecond
	requireCredentials := isProdEnv(cfg.Env)

	telegramBreaker := NewCircuitBreaker(failThreshold, successThreshold, openTimeout)
	slackBreaker := NewCircuitBreaker(failThreshold, successThreshold, openTimeout)
	smtpBreaker := NewCircuitBreaker(failThreshold, successThreshold, openTimeout)
	smsBreaker := NewCircuitBreaker(failThreshold, successThreshold, openTimeout)

	return ProviderBundle{
		Providers: map[db.NotifierProvider]Provider{
			db.NotifierProviderTELEGRAM: NewTelegramProvider(
				string(n.TelegramBotToken),
				n.TelegramChatID,
				telegramBreaker,
				requireCredentials,
			),
			db.NotifierProviderSLACK: NewSlackProvider(
				string(n.SlackWebhookURL),
				slackBreaker,
				requireCredentials,
			),
			db.NotifierProviderSMTP: NewSMTPProvider(
				n.SMTPHost,
				n.SMTPPort,
				n.SMTPUsername,
				string(n.SMTPPassword),
				n.SMTPSender,
				smtpBreaker,
				requireCredentials,
			),
			db.NotifierProviderSMS: NewSMSProvider(
				n.SMSProviderURL,
				string(n.SMSAPIToken),
				n.SMSDefaultRecipient,
				smsBreaker,
				requireCredentials,
			),
		},
		Breakers: map[db.NotifierProvider]*CircuitBreaker{
			db.NotifierProviderTELEGRAM: telegramBreaker,
			db.NotifierProviderSLACK:    slackBreaker,
			db.NotifierProviderSMTP:     smtpBreaker,
			db.NotifierProviderSMS:      smsBreaker,
		},
	}
}

func StartCircuitBreakerMetricsScraper(ctx context.Context, breakers map[db.NotifierProvider]*CircuitBreaker, interval time.Duration) {
	if len(breakers) == 0 {
		return
	}
	if interval <= 0 {
		interval = 15 * time.Second
	}

	scrape := func() {
		for provider, breaker := range breakers {
			if breaker == nil {
				continue
			}
			recordCircuitBreakerState(ProviderDisplayName(provider), breaker.State())
		}
	}

	scrape()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			scrape()
		}
	}
}
