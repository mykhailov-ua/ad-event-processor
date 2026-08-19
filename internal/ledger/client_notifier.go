package ledger

import (
	"context"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/notify"
)

func NewNotifierAPI(ctx context.Context, cfg *config.Config) (notify.NotifierAPI, func(), error) {
	if cfg == nil {
		return nil, func() {}, nil
	}
	_, recipient := ResolveInvoiceNotifierTarget(cfg)
	if recipient == "" {
		return nil, func() {}, nil
	}
	return notify.OpenAPI(ctx, cfg)
}

func ResolveInvoiceNotifierTarget(cfg *config.Config) (provider, recipient string) {
	if cfg == nil {
		return "", ""
	}
	if cfg.Notifier.InvoiceRecipient != "" {
		return notify.MapConfigProviderName(cfg.Notifier.InvoiceProvider), cfg.Notifier.InvoiceRecipient
	}
	if cfg.Notifier.TelegramChatID != "" {
		return notify.ProviderTelegram, cfg.Notifier.TelegramChatID
	}
	if cfg.Notifier.SlackWebhookURL != "" {
		return notify.ProviderSlack, string(cfg.Notifier.SlackWebhookURL)
	}
	if cfg.Notifier.SMTPSender != "" {
		return notify.ProviderSMTP, cfg.Notifier.SMTPSender
	}
	return "", ""
}
