package notify

import (
	"strings"

	"ad-event-processor/internal/config"
)

const (
	ProviderTelegram = "TELEGRAM"
	ProviderSlack    = "SLACK"
	ProviderSMS      = "SMS"
	ProviderSMTP     = "SMTP"
)

type OpsAlertTarget struct {
	Provider  string
	Recipient string
}

func MapConfigProviderName(raw string) string {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case ProviderSlack:
		return ProviderSlack
	case ProviderSMTP:
		return ProviderSMTP
	case ProviderSMS:
		return ProviderSMS
	default:
		return ProviderTelegram
	}
}

func ResolveOpsAlertTargets(cfg *config.Config) []OpsAlertTarget {
	if cfg == nil {
		return nil
	}

	var targets []OpsAlertTarget
	if cfg.Notifier.TelegramChatID != "" {
		targets = append(targets, OpsAlertTarget{
			Provider:  ProviderTelegram,
			Recipient: cfg.Notifier.TelegramChatID,
		})
	}
	if cfg.Notifier.SlackWebhookURL != "" {
		targets = append(targets, OpsAlertTarget{
			Provider:  ProviderSlack,
			Recipient: string(cfg.Notifier.SlackWebhookURL),
		})
	}
	if cfg.Notifier.SMSDefaultRecipient != "" {
		targets = append(targets, OpsAlertTarget{
			Provider:  ProviderSMS,
			Recipient: cfg.Notifier.SMSDefaultRecipient,
		})
	}
	if cfg.Notifier.SMTPSender != "" {
		targets = append(targets, OpsAlertTarget{
			Provider:  ProviderSMTP,
			Recipient: cfg.Notifier.SMTPSender,
		})
	}
	return targets
}

func ResolveOpsAlertTarget(cfg *config.Config) (provider, recipient string, ok bool) {
	targets := ResolveOpsAlertTargets(cfg)
	if len(targets) == 0 {
		return "", "", false
	}
	t := targets[0]
	return t.Provider, t.Recipient, true
}

func ResolveBroadcastProviders(cfg *config.Config) []string {
	targets := ResolveOpsAlertTargets(cfg)
	if len(targets) == 0 {
		return nil
	}
	providers := make([]string, 0, len(targets))
	for _, target := range targets {
		providers = append(providers, target.Provider)
	}
	return providers
}
