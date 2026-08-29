package settlement

import (
	"context"
	"sync"

	"ad-event-processor/internal/notify"
)

type stubNotifierAPI struct {
	mu     sync.Mutex
	inputs []notify.NotificationInput
}

func (na *stubNotifierAPI) SendNotification(
	_ context.Context,
	provider, recipient, title, body string,
) (notify.SendNotificationResult, error) {
	return na.SendNotificationInput(context.Background(), notify.NotificationInput{
		Provider:  provider,
		Recipient: recipient,
		Title:     title,
		Body:      body,
	})
}

func (na *stubNotifierAPI) SendNotificationInput(
	_ context.Context,
	input notify.NotificationInput,
) (notify.SendNotificationResult, error) {
	na.mu.Lock()
	defer na.mu.Unlock()
	na.inputs = append(na.inputs, input)
	return notify.SendNotificationResult{NotificationID: "stub-id"}, nil
}

func (na *stubNotifierAPI) SendNotificationBatch(
	_ context.Context,
	inputs []notify.NotificationInput,
) ([]notify.SendNotificationResult, error) {
	na.mu.Lock()
	defer na.mu.Unlock()
	na.inputs = append(na.inputs, inputs...)
	out := make([]notify.SendNotificationResult, len(inputs))
	for i := range inputs {
		out[i] = notify.SendNotificationResult{NotificationID: "stub-id"}
	}
	return out, nil
}

func (na *stubNotifierAPI) snapshot() []notify.NotificationInput {
	na.mu.Lock()
	defer na.mu.Unlock()
	out := make([]notify.NotificationInput, len(na.inputs))
	copy(out, na.inputs)
	return out
}
