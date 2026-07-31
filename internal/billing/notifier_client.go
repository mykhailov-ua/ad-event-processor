package billing

import (
	"context"

	"espx/internal/config"
	"espx/internal/notifier"
	notifierpb "espx/internal/notifier/pb"
	"strings"
)

func NewNotifierAPI(ctx context.Context, cfg *config.Config) (notifier.NotifierAPI, func(), error) {
	if cfg == nil {
		return nil, func() {}, nil
	}
	_, recipient := ResolveInvoiceNotifierTarget(cfg)
	if recipient == "" {
		return nil, func() {}, nil
	}
	return notifier.OpenAPIOrDial(ctx, cfg)
}

func ResolveInvoiceNotifierTarget(cfg *config.Config) (notifierpb.Provider, string) {
	if cfg == nil {
		return notifierpb.Provider_PROVIDER_UNSPECIFIED, ""
	}
	if cfg.Notifier.InvoiceRecipient != "" {
		return mapInvoiceProvider(cfg.Notifier.InvoiceProvider), cfg.Notifier.InvoiceRecipient
	}
	if cfg.Notifier.TelegramChatID != "" {
		return notifierpb.Provider_PROVIDER_TELEGRAM, cfg.Notifier.TelegramChatID
	}
	if cfg.Notifier.SlackWebhookURL != "" {
		return notifierpb.Provider_PROVIDER_SLACK, string(cfg.Notifier.SlackWebhookURL)
	}
	if cfg.Notifier.SMTPSender != "" {
		return notifierpb.Provider_PROVIDER_SMTP, cfg.Notifier.SMTPSender
	}
	return notifierpb.Provider_PROVIDER_UNSPECIFIED, ""
}

func mapInvoiceProvider(raw string) notifierpb.Provider {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "SLACK":
		return notifierpb.Provider_PROVIDER_SLACK
	case "SMTP":
		return notifierpb.Provider_PROVIDER_SMTP
	case "SMS":
		return notifierpb.Provider_PROVIDER_SMS
	default:
		return notifierpb.Provider_PROVIDER_TELEGRAM
	}
}
