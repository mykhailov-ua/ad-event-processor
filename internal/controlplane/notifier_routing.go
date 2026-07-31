package controlplane

import (
	"context"
	"fmt"
	"strings"

	"espx/internal/config"
	"espx/internal/notifier"
	notifierpb "espx/internal/notifier/pb"
)

type opsAlertTarget struct {
	Provider  notifierpb.Provider
	Recipient string
}

func resolveOpsAlertTarget(cfg *config.Config) (notifierpb.Provider, string, bool) {
	targets := resolveOpsAlertTargets(cfg)
	if len(targets) == 0 {
		return notifierpb.Provider_PROVIDER_UNSPECIFIED, "", false
	}
	primary := targets[0]
	return primary.Provider, primary.Recipient, true
}

func resolveOpsAlertTargets(cfg *config.Config) []opsAlertTarget {
	if cfg == nil {
		return nil
	}

	var targets []opsAlertTarget
	if cfg.Notifier.TelegramChatID != "" {
		targets = append(targets, opsAlertTarget{
			Provider:  notifierpb.Provider_PROVIDER_TELEGRAM,
			Recipient: cfg.Notifier.TelegramChatID,
		})
	}
	if cfg.Notifier.SlackWebhookURL != "" {
		targets = append(targets, opsAlertTarget{
			Provider:  notifierpb.Provider_PROVIDER_SLACK,
			Recipient: string(cfg.Notifier.SlackWebhookURL),
		})
	}
	if cfg.Notifier.SMSDefaultRecipient != "" {
		targets = append(targets, opsAlertTarget{
			Provider:  notifierpb.Provider_PROVIDER_SMS,
			Recipient: cfg.Notifier.SMSDefaultRecipient,
		})
	}
	if cfg.Notifier.SMTPSender != "" {
		targets = append(targets, opsAlertTarget{
			Provider:  notifierpb.Provider_PROVIDER_SMTP,
			Recipient: cfg.Notifier.SMTPSender,
		})
	}
	return targets
}

func resolveBroadcastProviders(cfg *config.Config) []notifierpb.Provider {
	targets := resolveOpsAlertTargets(cfg)
	if len(targets) == 0 {
		return nil
	}
	providers := make([]notifierpb.Provider, 0, len(targets))
	for _, target := range targets {
		providers = append(providers, target.Provider)
	}
	return providers
}

func alertSeverityBroadcast(alert AlertmanagerAlert) bool {
	severity := strings.ToLower(strings.TrimSpace(alert.Labels["severity"]))
	return severity == "critical"
}

func enqueueOpsNotification(
	ctx context.Context,
	client *NotifierClient,
	target opsAlertTarget,
	title, body, dedupKey string,
	broadcast bool,
	broadcastProviders []notifierpb.Provider,
) error {
	if client == nil || client.api == nil {
		return fmt.Errorf("notifier client not configured")
	}

	broadcastProviderNames := make([]string, 0, len(broadcastProviders))
	for _, p := range broadcastProviders {
		broadcastProviderNames = append(broadcastProviderNames, p.String())
	}
	_, err := client.api.SendNotificationInput(ctx, notifier.NotificationInput{
		Provider:           target.Provider.String(),
		Recipient:          target.Recipient,
		Title:              title,
		Body:               body,
		DedupKey:           dedupKey,
		Broadcast:          broadcast,
		BroadcastProviders: broadcastProviderNames,
	})
	return err
}

func enqueueOpsNotificationBatch(
	ctx context.Context,
	client *NotifierClient,
	items []opsNotificationItem,
) error {
	if client == nil || client.api == nil {
		return fmt.Errorf("notifier client not configured")
	}
	if len(items) == 0 {
		return nil
	}

	inputs := make([]notifier.NotificationInput, 0, len(items))
	for _, item := range items {
		broadcastProviderNames := make([]string, 0, len(item.BroadcastProviders))
		for _, p := range item.BroadcastProviders {
			broadcastProviderNames = append(broadcastProviderNames, p.String())
		}
		inputs = append(inputs, notifier.NotificationInput{
			Provider:           item.Target.Provider.String(),
			Recipient:          item.Target.Recipient,
			Title:              item.Title,
			Body:               item.Body,
			DedupKey:           item.DedupKey,
			Broadcast:          item.Broadcast,
			BroadcastProviders: broadcastProviderNames,
		})
	}

	_, err := client.api.SendNotificationBatch(ctx, inputs)
	return err
}

type opsNotificationItem struct {
	Target             opsAlertTarget
	Title              string
	Body               string
	DedupKey           string
	Broadcast          bool
	BroadcastProviders []notifierpb.Provider
}
