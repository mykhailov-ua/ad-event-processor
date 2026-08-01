package paymenttest

import (
	"context"
	"sync"

	"espx/internal/config"
	"espx/internal/notify"
)

type StubNotifierAPI struct {
	mu     sync.Mutex
	inputs []notify.NotificationInput
}

func (stub *StubNotifierAPI) SendNotification(
	_ context.Context,
	provider, recipient, title, body string,
) (notify.SendNotificationResult, error) {
	return stub.SendNotificationInput(context.Background(), notify.NotificationInput{
		Provider:  provider,
		Recipient: recipient,
		Title:     title,
		Body:      body,
	})
}

func (stub *StubNotifierAPI) SendNotificationInput(
	_ context.Context,
	input notify.NotificationInput,
) (notify.SendNotificationResult, error) {
	stub.mu.Lock()
	defer stub.mu.Unlock()
	stub.inputs = append(stub.inputs, input)
	return notify.SendNotificationResult{NotificationID: "stub-id"}, nil
}

func (stub *StubNotifierAPI) SendNotificationBatch(
	_ context.Context,
	inputs []notify.NotificationInput,
) ([]notify.SendNotificationResult, error) {
	stub.mu.Lock()
	defer stub.mu.Unlock()
	stub.inputs = append(stub.inputs, inputs...)
	out := make([]notify.SendNotificationResult, len(inputs))
	for i := range inputs {
		out[i] = notify.SendNotificationResult{NotificationID: "stub-id"}
	}
	return out, nil
}

func (stub *StubNotifierAPI) Snapshot() []notify.NotificationInput {
	stub.mu.Lock()
	defer stub.mu.Unlock()
	out := make([]notify.NotificationInput, len(stub.inputs))
	copy(out, stub.inputs)
	return out
}

func TestOpsConfig() *config.Config {
	cfg := &config.Config{}
	cfg.Management.OpsAlertsEnabled = true
	cfg.Notifier.TelegramChatID = "-100123"
	return cfg
}
