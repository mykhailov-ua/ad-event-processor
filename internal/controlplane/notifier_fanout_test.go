package controlplane

import (
	"espx/pkg/faultproof"

	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"espx/internal/config"
	"espx/internal/notify"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testNotifierClient(stub notify.NotifierAPI) *NotifierClient {
	return &NotifierClient{api: stub}
}

type stubNotifierAPITest struct {
	mu     sync.Mutex
	inputs []notify.NotificationInput
	fail   bool
}

func (stub *stubNotifierAPITest) SendNotification(
	ctx context.Context,
	provider, recipient, title, body string,
) (notify.SendNotificationResult, error) {
	return stub.SendNotificationInput(ctx, notify.NotificationInput{
		Provider:  provider,
		Recipient: recipient,
		Title:     title,
		Body:      body,
	})
}

func (stub *stubNotifierAPITest) SendNotificationInput(
	ctx context.Context,
	input notify.NotificationInput,
) (notify.SendNotificationResult, error) {
	stub.mu.Lock()
	defer stub.mu.Unlock()
	if stub.fail {
		return notify.SendNotificationResult{}, fmt.Errorf("stub notifier unavailable")
	}
	stub.inputs = append(stub.inputs, input)
	return notify.SendNotificationResult{NotificationID: "stub-id"}, nil
}

func (stub *stubNotifierAPITest) SendNotificationBatch(
	ctx context.Context,
	inputs []notify.NotificationInput,
) ([]notify.SendNotificationResult, error) {
	stub.mu.Lock()
	defer stub.mu.Unlock()
	if stub.fail {
		return nil, fmt.Errorf("stub notifier unavailable")
	}
	stub.inputs = append(stub.inputs, inputs...)
	out := make([]notify.SendNotificationResult, len(inputs))
	for i := range inputs {
		out[i] = notify.SendNotificationResult{NotificationID: "stub-id"}
	}
	return out, nil
}

func (stub *stubNotifierAPITest) snapshot() []notify.NotificationInput {
	stub.mu.Lock()
	defer stub.mu.Unlock()
	out := make([]notify.NotificationInput, len(stub.inputs))
	copy(out, stub.inputs)
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
	targets := resolveOpsAlertTargets(testNotifierConfig())
	require.Len(t, targets, 3)
	assert.Equal(t, notify.ProviderTelegram, targets[0].Provider)
	assert.Equal(t, notify.ProviderSlack, targets[1].Provider)
	assert.Equal(t, notify.ProviderSMS, targets[2].Provider)
}

func TestResolveBroadcastProviders_AllConfigured(t *testing.T) {
	providers := resolveBroadcastProviders(testNotifierConfig())
	require.Len(t, providers, 3)
	assert.Equal(t, notify.ProviderTelegram, providers[0])
	assert.Equal(t, notify.ProviderSlack, providers[1])
	assert.Equal(t, notify.ProviderSMS, providers[2])
}

func TestAlertSeverityBroadcast(t *testing.T) {
	assert.True(t, alertSeverityBroadcast(AlertmanagerAlert{
		Labels: map[string]string{"severity": "critical"},
	}))
	assert.False(t, alertSeverityBroadcast(AlertmanagerAlert{
		Labels: map[string]string{"severity": "warning"},
	}))
	assert.False(t, alertSeverityBroadcast(AlertmanagerAlert{
		Labels: map[string]string{},
	}))
}

func TestAlertmanagerWebhook_CriticalUsesBroadcast(t *testing.T) {
	stub := &stubNotifierAPITest{}
	cfg := testNotifierConfig()
	cfg.Management.AlertmanagerWebhookEnabled = true

	h := &AlertmanagerWebhook{
		client:             testNotifierClient(stub),
		provider:           notify.ProviderTelegram,
		recipient:          cfg.Notifier.TelegramChatID,
		broadcastProviders: resolveBroadcastProviders(cfg),
	}

	payload := AlertmanagerPayload{
		Alerts: []AlertmanagerAlert{{
			Status: "firing",
			Labels: map[string]string{
				"alertname": "HighErrorRate",
				"severity":  "critical",
			},
			Annotations: map[string]string{
				"summary":     "Tracker errors elevated",
				"description": "5xx ratio above SLO",
			},
			StartsAt: time.Now().UTC(),
		}},
	}
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/ops/alertmanager/webhook", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.handle(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	requests := stub.snapshot()
	require.Len(t, requests, 1)
	assert.True(t, requests[0].Broadcast)
	assert.Len(t, requests[0].BroadcastProviders, 3)
	assert.Equal(t, "alertmanager:HighErrorRate:firing", requests[0].DedupKey)
}

func TestAlertmanagerWebhook_WarningUsesFallback(t *testing.T) {
	stub := &stubNotifierAPITest{}
	cfg := testNotifierConfig()

	h := &AlertmanagerWebhook{
		client:             testNotifierClient(stub),
		provider:           notify.ProviderTelegram,
		recipient:          cfg.Notifier.TelegramChatID,
		broadcastProviders: resolveBroadcastProviders(cfg),
	}

	payload := AlertmanagerPayload{
		Alerts: []AlertmanagerAlert{{
			Status: "firing",
			Labels: map[string]string{
				"alertname": "LogCompactorHotLagHigh",
				"severity":  "warning",
			},
			Annotations: map[string]string{"summary": "Hot lag high"},
			StartsAt:    time.Now().UTC(),
		}},
	}
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/ops/alertmanager/webhook", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.handle(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	requests := stub.snapshot()
	require.Len(t, requests, 1)
	assert.False(t, requests[0].Broadcast)
	assert.Empty(t, requests[0].BroadcastProviders)
}

func TestOpsAlerter_CriticalEventBroadcast(t *testing.T) {
	stub := &stubNotifierAPITest{}
	cfg := testNotifierConfig()
	cfg.Management.OpsAlertsEnabled = true

	alerter := NewOpsAlerter(testNotifierClient(stub), cfg)
	require.NotNil(t, alerter)

	err := alerter.enqueueNotification(context.Background(), "recon", "recon", "body", true)
	require.NoError(t, err)

	requests := stub.snapshot()
	require.Len(t, requests, 1)
	assert.True(t, requests[0].Broadcast)
	assert.Len(t, requests[0].BroadcastProviders, 3)
}

func TestOpsAlerter_WarningEventFallback(t *testing.T) {
	stub := &stubNotifierAPITest{}
	cfg := testNotifierConfig()
	cfg.Management.OpsAlertsEnabled = true

	alerter := NewOpsAlerter(testNotifierClient(stub), cfg)
	require.NotNil(t, alerter)

	err := alerter.enqueueNotification(context.Background(), "migration", "migration", "body", false)
	require.NoError(t, err)

	requests := stub.snapshot()
	require.Len(t, requests, 1)
	assert.False(t, requests[0].Broadcast)
}

func TestFault_alertmanagerWebhookFanOut(t *testing.T) {
	stub := &stubNotifierAPITest{}
	cfg := testNotifierConfig()
	cfg.Management.AlertmanagerWebhookEnabled = true

	h := &AlertmanagerWebhook{
		client:             testNotifierClient(stub),
		provider:           notify.ProviderTelegram,
		recipient:          cfg.Notifier.TelegramChatID,
		broadcastProviders: resolveBroadcastProviders(cfg),
	}

	const alertCount = 3
	alerts := make([]AlertmanagerAlert, 0, alertCount)
	for i := range alertCount {
		alerts = append(alerts, AlertmanagerAlert{
			Status: "firing",
			Labels: map[string]string{
				"alertname": "RedisInstanceDown",
				"severity":  "critical",
			},
			Annotations: map[string]string{"summary": "Redis down"},
			StartsAt:    time.Now().UTC(),
		})
		_ = i
	}

	payload := AlertmanagerPayload{Alerts: alerts}
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/ops/alertmanager/webhook", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.handle(rec, req)

	requests := stub.snapshot()
	require.Len(t, requests, alertCount)
	for _, req := range requests {
		assert.True(t, req.Broadcast)
		assert.Len(t, req.BroadcastProviders, 3)
	}

	faultproof.Log(t, "alertmanager_webhook_fanout", map[string]string{
		"alerts":   "3",
		"channels": "3",
		"mode":     "BROADCAST",
		"severity": "critical",
	})
}

func TestFault_opsEventFanOut(t *testing.T) {
	stub := &stubNotifierAPITest{}
	cfg := testNotifierConfig()
	cfg.Management.OpsAlertsEnabled = true

	alerter := NewOpsAlerter(testNotifierClient(stub), cfg)
	require.NotNil(t, alerter)

	alerter.AlertReconDiscrepancy(42, 3, 1000, "2026-07-04")
	alerter.AlertRedisShardUnhealthy(1, assert.AnError)
	alerter.AlertDrainStuck(7, 3, "draining", "timeout", time.Now().UTC())
	alerter.AlertSlotMapMigrating(2, []int16{1, 2}, 0)

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
