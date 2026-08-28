package opsadmin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/notify"
	"ad-event-processor/pkg/faultproof"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubNotifierAPI struct {
	mu     sync.Mutex
	inputs []notify.NotificationInput
	fail   bool
}

func (na *stubNotifierAPI) SendNotification(
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

func (na *stubNotifierAPI) SendNotificationInput(
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

func (na *stubNotifierAPI) SendNotificationBatch(
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

func (na *stubNotifierAPI) snapshot() []notify.NotificationInput {
	na.mu.Lock()
	defer na.mu.Unlock()
	out := make([]notify.NotificationInput, len(na.inputs))
	copy(out, na.inputs)
	return out
}

func testAlertmanagerConfig() *config.Config {
	cfg := &config.Config{}
	cfg.Env = "production"
	cfg.Notifier.TelegramChatID = "-100123"
	cfg.Notifier.SlackWebhookURL = "https://hooks.slack.com/services/test"
	cfg.Notifier.SMSDefaultRecipient = "+79001234567"
	return cfg
}

func postAlertmanagerWebhook(t *testing.T, h *AlertmanagerWebhook, payload AlertmanagerPayload) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(payload)
	require.NoError(t, err)
	mux := http.NewServeMux()
	h.Register(mux)
	req := httptest.NewRequest(http.MethodPost, "/ops/alertmanager/webhook", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestFormatAlertmanagerAlert_Active(t *testing.T) {
	start := time.Date(2026, 7, 4, 12, 0, 0, 0, time.UTC)
	_, body := FormatAlertmanagerAlert(AlertmanagerAlert{
		Status: "firing",
		Labels: map[string]string{
			"alertname": "HighErrorRate",
			"severity":  "critical",
		},
		Annotations: map[string]string{
			"summary":     "Tracker errors elevated",
			"description": "5xx ratio above SLO",
		},
		StartsAt: start,
	})

	for _, want := range []string{"ALERT ACTIVE", "critical", "Tracker errors elevated", "5xx ratio above SLO"} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing %q: %s", want, body)
		}
	}
}

func TestFormatAlertmanagerAlert_Resolved(t *testing.T) {
	_, body := FormatAlertmanagerAlert(AlertmanagerAlert{
		Status: "resolved",
		Labels: map[string]string{"severity": "warning"},
		Annotations: map[string]string{
			"summary": "Recovered",
		},
		StartsAt: time.Now().UTC(),
	})
	if !strings.Contains(body, "ALERT RESOLVED") {
		t.Fatalf("expected resolved marker in %q", body)
	}
}

func TestNewAlertmanagerWebhook_DisabledByDefault(t *testing.T) {
	cfg := &config.Config{}
	cfg.Management.AlertmanagerWebhookEnabled = false
	if NewAlertmanagerWebhook(&stubNotifierAPI{}, cfg) != nil {
		t.Fatal("expected nil when disabled")
	}
}

func TestNewAlertmanagerWebhook_EnabledWithRecipient(t *testing.T) {
	cfg := testAlertmanagerConfig()
	cfg.Management.AlertmanagerWebhookEnabled = true
	h := NewAlertmanagerWebhook(&stubNotifierAPI{}, cfg)
	if h == nil {
		t.Fatal("expected handler")
	}
	if h.AlertmanagerRecipient() != "-100123" {
		t.Fatalf("recipient: got %q", h.AlertmanagerRecipient())
	}
}

func TestAlertmanagerWebhook_CriticalUsesBroadcast(t *testing.T) {
	stub := &stubNotifierAPI{}
	cfg := testAlertmanagerConfig()
	cfg.Management.AlertmanagerWebhookEnabled = true

	h := NewAlertmanagerWebhook(stub, cfg)
	require.NotNil(t, h)

	rec := postAlertmanagerWebhook(t, h, AlertmanagerPayload{
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
	})

	assert.Equal(t, http.StatusOK, rec.Code)
	requests := stub.snapshot()
	require.Len(t, requests, 1)
	assert.True(t, requests[0].Broadcast)
	assert.Len(t, requests[0].BroadcastProviders, 3)
	assert.Equal(t, "alertmanager:HighErrorRate:firing", requests[0].DedupKey)
}

func TestAlertmanagerWebhook_WarningUsesFallback(t *testing.T) {
	stub := &stubNotifierAPI{}
	cfg := testAlertmanagerConfig()
	cfg.Management.AlertmanagerWebhookEnabled = true

	h := NewAlertmanagerWebhook(stub, cfg)
	require.NotNil(t, h)

	rec := postAlertmanagerWebhook(t, h, AlertmanagerPayload{
		Alerts: []AlertmanagerAlert{{
			Status: "firing",
			Labels: map[string]string{
				"alertname": "LogCompactorHotLagHigh",
				"severity":  "warning",
			},
			Annotations: map[string]string{"summary": "Hot lag high"},
			StartsAt:    time.Now().UTC(),
		}},
	})

	assert.Equal(t, http.StatusOK, rec.Code)
	requests := stub.snapshot()
	require.Len(t, requests, 1)
	assert.False(t, requests[0].Broadcast)
	assert.Empty(t, requests[0].BroadcastProviders)
}

func TestFault_alertmanagerWebhookFanOut(t *testing.T) {
	stub := &stubNotifierAPI{}
	cfg := testAlertmanagerConfig()
	cfg.Management.AlertmanagerWebhookEnabled = true

	h := NewAlertmanagerWebhook(stub, cfg)
	require.NotNil(t, h)

	const alertCount = 3
	alerts := make([]AlertmanagerAlert, 0, alertCount)
	for range alertCount {
		alerts = append(alerts, AlertmanagerAlert{
			Status: "firing",
			Labels: map[string]string{
				"alertname": "RedisInstanceDown",
				"severity":  "critical",
			},
			Annotations: map[string]string{"summary": "Redis down"},
			StartsAt:    time.Now().UTC(),
		})
	}

	rec := postAlertmanagerWebhook(t, h, AlertmanagerPayload{Alerts: alerts})
	assert.Equal(t, http.StatusOK, rec.Code)

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

func TestAlertSeverityBroadcast(t *testing.T) {
	assert.True(t, AlertSeverityBroadcast(AlertmanagerAlert{
		Labels: map[string]string{"severity": "critical"},
	}))
	assert.False(t, AlertSeverityBroadcast(AlertmanagerAlert{
		Labels: map[string]string{"severity": "warning"},
	}))
	assert.False(t, AlertSeverityBroadcast(AlertmanagerAlert{
		Labels: map[string]string{},
	}))
}
