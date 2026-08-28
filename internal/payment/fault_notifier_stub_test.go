package payment_test

import (
	"context"
	"sync"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/notify"
)

type StubNotifierAPI struct {
	mu     sync.Mutex
	inputs []notify.NotificationInput
}

func (sn *StubNotifierAPI) SendNotification(
	_ context.Context,
	provider, recipient, title, body string,
) (notify.SendNotificationResult, error) {
	return sn.SendNotificationInput(context.Background(), notify.NotificationInput{
		Provider:  provider,
		Recipient: recipient,
		Title:     title,
		Body:      body,
	})
}

func (sn *StubNotifierAPI) SendNotificationInput(
	_ context.Context,
	input notify.NotificationInput,
) (notify.SendNotificationResult, error) {
	sn.mu.Lock()
	defer sn.mu.Unlock()
	sn.inputs = append(sn.inputs, input)
	return notify.SendNotificationResult{NotificationID: "stub-id"}, nil
}

func (sn *StubNotifierAPI) SendNotificationBatch(
	_ context.Context,
	inputs []notify.NotificationInput,
) ([]notify.SendNotificationResult, error) {
	sn.mu.Lock()
	defer sn.mu.Unlock()
	sn.inputs = append(sn.inputs, inputs...)
	out := make([]notify.SendNotificationResult, len(inputs))
	for i := range inputs {
		out[i] = notify.SendNotificationResult{NotificationID: "stub-id"}
	}
	return out, nil
}

func (sn *StubNotifierAPI) Snapshot() []notify.NotificationInput {
	sn.mu.Lock()
	defer sn.mu.Unlock()
	out := make([]notify.NotificationInput, len(sn.inputs))
	copy(out, sn.inputs)
	return out
}

func faultTestOpsConfig() *config.Config {
	cfg := &config.Config{}
	cfg.Management.OpsAlertsEnabled = true
	cfg.Notifier.TelegramChatID = "-100123"
	return cfg
}
