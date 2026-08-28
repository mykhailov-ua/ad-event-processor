package controlplane

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"ad-event-processor/pkg/faultproof"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/notify"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubNotifierAPITest struct {
	mu     sync.Mutex
	inputs []notify.NotificationInput
	fail   bool
}

func (na *stubNotifierAPITest) SendNotification(
	ctx context.Context,
	provider, recipient, title, body string,
) (notify.SendNotificationResult, error) {
	return na.SendNotificationInput(ctx, notify.NotificationInput{
		Provider:  provider,
		Recipient: recipient,
		Title:     title,
		Body:      body,
	})
}

func (na *stubNotifierAPITest) SendNotificationInput(
	ctx context.Context,
	input notify.NotificationInput,
) (notify.SendNotificationResult, error) {
	na.mu.Lock()
	defer na.mu.Unlock()
	if na.fail {
		return notify.SendNotificationResult{}, fmt.Errorf("stub notifier unavailable")
	}
	na.inputs = append(na.inputs, input)
	return notify.SendNotificationResult{NotificationID: "stub-id"}, nil
}

func (na *stubNotifierAPITest) SendNotificationBatch(
	ctx context.Context,
	inputs []notify.NotificationInput,
) ([]notify.SendNotificationResult, error) {
	na.mu.Lock()
	defer na.mu.Unlock()
	if na.fail {
		return nil, fmt.Errorf("stub notifier unavailable")
	}
	na.inputs = append(na.inputs, inputs...)
	out := make([]notify.SendNotificationResult, len(inputs))
	for i := range inputs {
		out[i] = notify.SendNotificationResult{NotificationID: "stub-id"}
	}
	return out, nil
}

func (na *stubNotifierAPITest) snapshot() []notify.NotificationInput {
	na.mu.Lock()
	defer na.mu.Unlock()
	out := make([]notify.NotificationInput, len(na.inputs))
	copy(out, na.inputs)
	return out
}

func testNotifierConfig() *config.Config {
	cfg := &config.Config{}
	cfg.Env = "production"
	cfg.Notifier.TelegramChatID = "-100123"
	cfg.Notifier.SlackWebhookURL = "https://hooks.slack.com/services/test"
	cfg.Notifier.SMSDefaultRecipient = "+79001234567"
	return cfg
}

func TestResolveOpsAlertTargets_MultiChannel(t *testing.T) {
	targets := notify.ResolveOpsAlertTargets(testNotifierConfig())
	require.Len(t, targets, 3)
	assert.Equal(t, notify.ProviderTelegram, targets[0].Provider)
	assert.Equal(t, notify.ProviderSlack, targets[1].Provider)
	assert.Equal(t, notify.ProviderSMS, targets[2].Provider)
}

func TestResolveBroadcastProviders_AllConfigured(t *testing.T) {
	providers := notify.ResolveBroadcastProviders(testNotifierConfig())
	require.Len(t, providers, 3)
	assert.Equal(t, notify.ProviderTelegram, providers[0])
	assert.Equal(t, notify.ProviderSlack, providers[1])
	assert.Equal(t, notify.ProviderSMS, providers[2])
}

func TestFault_opsEventFanOut(t *testing.T) {
	stub := &stubNotifierAPITest{}
	cfg := testNotifierConfig()
	cfg.Management.OpsAlertsEnabled = true

	alerter := NewOpsAlerter(stub, cfg)
	require.NotNil(t, alerter)

	alerter.AlertReconDiscrepancy(context.Background(), 42, 3, 1000, "2026-07-04")
	alerter.AlertRedisShardUnhealthy(context.Background(), 1, assert.AnError)
	alerter.AlertDrainStuck(context.Background(), 7, 3, "draining", "timeout", time.Now().UTC())
	alerter.AlertSlotMapMigrating(context.Background(), 2, []int16{1, 2}, 0)

	time.Sleep(100 * time.Millisecond)

	requests := stub.snapshot()
	require.Len(t, requests, 4)

	broadcastCount := 0
	fallbackCount := 0
	for _, req := range requests {
		if req.Broadcast {
			broadcastCount++
			assert.Len(t, req.BroadcastProviders, 3)
			continue
		}
		fallbackCount++
	}
	assert.Equal(t, 3, broadcastCount)
	assert.Equal(t, 1, fallbackCount)

	faultproof.Log(t, "ops_event_fanout", map[string]string{
		"critical_events": "3",
		"warning_events":  "1",
		"channels":        "3",
		"broadcast_mode":  "true",
	})
}
