package opsadmin

import (
	"context"
	"fmt"

	"ad-event-processor/internal/notify"
)

type OpsNotificationItem struct {
	Target             notify.OpsAlertTarget
	Title              string
	Body               string
	DedupKey           string
	Broadcast          bool
	BroadcastProviders []string
}

func EnqueueOpsNotification(
	ctx context.Context,
	api notify.NotifierAPI,
	target notify.OpsAlertTarget,
	title, body, dedupKey string,
	broadcast bool,
	broadcastProviders []string,
) error {
	if api == nil {
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
	_, err := api.SendNotificationInput(ctx, input)
	return err
}

func EnqueueOpsNotificationBatch(
	ctx context.Context,
	api notify.NotifierAPI,
	items []OpsNotificationItem,
) error {
	if api == nil {
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

	_, err := api.SendNotificationBatch(ctx, inputs)
	return err
}
