package payment

import (
	"context"
	"fmt"

	"espx/internal/config"
	"espx/internal/notifier"
)

func resolveOpsAlertTarget(cfg *config.Config) (string, string, bool) {
	return notifier.ResolveOpsAlertTarget(cfg)
}

func resolveBroadcastProviders(cfg *config.Config) []string {
	return notifier.ResolveBroadcastProviders(cfg)
}

func enqueueOpsNotification(
	ctx context.Context,
	client *NotifierClient,
	provider, recipient, title, body, dedupKey string,
	broadcast bool,
	broadcastProviders []string,
) error {
	if client == nil || client.api == nil {
		return fmt.Errorf("notifier client not configured")
	}
	input := notifier.NotificationInput{
		Provider:  provider,
		Recipient: recipient,
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
