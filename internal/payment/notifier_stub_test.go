package payment

import (
	"context"
	"sync"

	"espx/internal/notifier"
)

type stubNotifierAPI struct {
	mu     sync.Mutex
	inputs []notifier.NotificationInput
}

func (stub *stubNotifierAPI) SendNotification(
	_ context.Context,
	provider, recipient, title, body string,
) (notifier.SendNotificationResult, error) {
	return stub.SendNotificationInput(context.Background(), notifier.NotificationInput{
		Provider:  provider,
		Recipient: recipient,
		Title:     title,
		Body:      body,
	})
}

func (stub *stubNotifierAPI) SendNotificationInput(
	_ context.Context,
	input notifier.NotificationInput,
) (notifier.SendNotificationResult, error) {
	stub.mu.Lock()
	defer stub.mu.Unlock()
	stub.inputs = append(stub.inputs, input)
	return notifier.SendNotificationResult{NotificationID: "stub-id"}, nil
}

func (stub *stubNotifierAPI) SendNotificationBatch(
	_ context.Context,
	inputs []notifier.NotificationInput,
) ([]notifier.SendNotificationResult, error) {
	stub.mu.Lock()
	defer stub.mu.Unlock()
	stub.inputs = append(stub.inputs, inputs...)
	out := make([]notifier.SendNotificationResult, len(inputs))
	for i := range inputs {
		out[i] = notifier.SendNotificationResult{NotificationID: "stub-id"}
	}
	return out, nil
}

func (stub *stubNotifierAPI) snapshot() []notifier.NotificationInput {
	stub.mu.Lock()
	defer stub.mu.Unlock()
	out := make([]notifier.NotificationInput, len(stub.inputs))
	copy(out, stub.inputs)
	return out
}
