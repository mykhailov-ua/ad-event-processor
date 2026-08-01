package billing

import (
	"context"

	"espx/internal/config"
	"espx/internal/notifier"
)

func NewNotifierAPI(ctx context.Context, cfg *config.Config) (notifier.NotifierAPI, func(), error) {
	if cfg == nil {
		return nil, func() {}, nil
	}
	_, recipient := ResolveInvoiceNotifierTarget(cfg)
	if recipient == "" {
		return nil, func() {}, nil
	}
	return notifier.OpenAPI(ctx, cfg)
}

func ResolveInvoiceNotifierTarget(cfg *config.Config) (string, string) {
	if cfg == nil {
		return "", ""
	}
	if cfg.Notifier.InvoiceRecipient != "" {
		return notifier.MapConfigProviderName(cfg.Notifier.InvoiceProvider), cfg.Notifier.InvoiceRecipient
	}
	if cfg.Notifier.TelegramChatID != "" {
		return notifier.ProviderTelegram, cfg.Notifier.TelegramChatID
	}
	if cfg.Notifier.SlackWebhookURL != "" {
		return notifier.ProviderSlack, string(cfg.Notifier.SlackWebhookURL)
	}
	if cfg.Notifier.SMTPSender != "" {
		return notifier.ProviderSMTP, cfg.Notifier.SMTPSender
	}
	return "", ""
}
