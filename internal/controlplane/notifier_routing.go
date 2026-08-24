package controlplane

import (
	"context"
	"fmt"
	"strings"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/notify"
)

func resolveOpsAlertTarget(cfg *config.Config) (string, string, bool) {
	return notify.ResolveOpsAlertTarget(cfg)
}

func resolveOpsAlertTargets(cfg *config.Config) []notify.OpsAlertTarget {
	return notify.ResolveOpsAlertTargets(cfg)
}

func resolveBroadcastProviders(cfg *config.Config) []string {
	return notify.ResolveBroadcastProviders(cfg)
}

func alertSeverityBroadcast(alert AlertmanagerAlert) bool {
	severity := strings.ToLower(strings.TrimSpace(alert.Labels["severity"]))
	return severity == "critical"
}

func enqueueOpsNotification(
	ctx context.Context,
	client *NotifierClient,
	target notify.OpsAlertTarget,
	title, body, dedupKey string,
	broadcast bool,
	broadcastProviders []string,
) error {
	if client == nil || client.api == nil {
		return fmt.Errorf("notifier client not configured")
	}

	input := notify.NotificationInput{
		Provider:  target.Provider,
		Recipient: target.Recipient,
		Title:     title,
		Body:      body,
		DedupKey:  dedupKey,
		Broadcast: broadcast,
	}
	if broadcast {
		input.BroadcastProviders = broadcastProviders
	}
	_, err := client.api.SendNotificationInput(ctx, input)
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

	inputs := make([]notify.NotificationInput, 0, len(items))
	for _, item := range items {
		input := notify.NotificationInput{
			Provider:  item.Target.Provider,
			Recipient: item.Target.Recipient,
			Title:     item.Title,
			Body:      item.Body,
			DedupKey:  item.DedupKey,
			Broadcast: item.Broadcast,
		}
		if item.Broadcast {
			input.BroadcastProviders = item.BroadcastProviders
		}
		inputs = append(inputs, input)
	}

	_, err := client.api.SendNotificationBatch(ctx, inputs)
	return err
}

type opsNotificationItem struct {
	Target             notify.OpsAlertTarget
	Title              string
	Body               string
	DedupKey           string
	Broadcast          bool
	BroadcastProviders []string
}
