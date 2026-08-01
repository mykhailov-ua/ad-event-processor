package paymenttest

import (
	"context"
	"sync"

	"espx/internal/config"
	"espx/internal/notifier"
)

type StubNotifierAPI struct {
	mu     sync.Mutex
	inputs []notifier.NotificationInput
}

func (stub *StubNotifierAPI) SendNotification(
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

func (stub *StubNotifierAPI) SendNotificationInput(
	_ context.Context,
	input notifier.NotificationInput,
) (notifier.SendNotificationResult, error) {
	stub.mu.Lock()
	defer stub.mu.Unlock()
	stub.inputs = append(stub.inputs, input)
	return notifier.SendNotificationResult{NotificationID: "stub-id"}, nil
}

func (stub *StubNotifierAPI) SendNotificationBatch(
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

func (stub *StubNotifierAPI) Snapshot() []notifier.NotificationInput {
	stub.mu.Lock()
	defer stub.mu.Unlock()
	out := make([]notifier.NotificationInput, len(stub.inputs))
	copy(out, stub.inputs)
	return out
}

func TestOpsConfig() *config.Config {
	cfg := &config.Config{}
	cfg.Management.OpsAlertsEnabled = true
	cfg.Notifier.TelegramChatID = "-100123"
	return cfg
}
