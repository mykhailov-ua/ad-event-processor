package payment

import (
	"context"
	"fmt"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/notify"
)

func resolveOpsAlertTarget(cfg *config.Config) (string, string, bool) {
	return notify.ResolveOpsAlertTarget(cfg)
}

func resolveBroadcastProviders(cfg *config.Config) []string {
	return notify.ResolveBroadcastProviders(cfg)
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
	input := notify.NotificationInput{
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
