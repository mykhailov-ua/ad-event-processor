package opsadmin

import (
	"context"
	"strings"
	"testing"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/notify"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testOpsAlerterConfig() *config.Config {
	cfg := &config.Config{}
	cfg.Env = "production"
	cfg.Management.OpsAlertsEnabled = true
	cfg.Notifier.TelegramChatID = "-100123"
	cfg.Notifier.SlackWebhookURL = "https://hooks.slack.com/services/test"
	cfg.Notifier.SMSDefaultRecipient = "+79001234567"
	return cfg
}

func TestResolveOpsAlertTarget_TelegramPreferred(t *testing.T) {
	cfg := &config.Config{}
	cfg.Notifier.TelegramChatID = "-100123"
	cfg.Notifier.SlackWebhookURL = "https://hooks.slack.com/test"

	provider, recipient, ok := ResolveOpsAlertTarget(cfg)
	if !ok {
		t.Fatal("expected target")
	}
	if provider != notify.ProviderTelegram {
		t.Fatalf("provider: got %v want TELEGRAM", provider)
	}
	if recipient != "-100123" {
		t.Fatalf("recipient: got %q", recipient)
	}
}

func TestOpsAlerter_CooldownDedup(t *testing.T) {
	cfg := testOpsAlerterConfig()
	cfg.Management.OpsAlertCooldownSec = 60

	alerter := NewOpsAlerter(&stubNotifierAPI{}, cfg)
	if alerter == nil {
		t.Fatal("expected alerter")
	}

	if !alerter.shouldSend("redis:shard:0") {
		t.Fatal("first send should pass")
	}
	if alerter.shouldSend("redis:shard:0") {
		t.Fatal("second send within cooldown should be suppressed")
	}

	alerter.lastSent.Store("redis:shard:0", time.Now().Add(-2*time.Minute))
	if !alerter.shouldSend("redis:shard:0") {
		t.Fatal("send after cooldown should pass")
	}
}

func TestNewOpsAlerter_DisabledWithoutRecipient(t *testing.T) {
	cfg := &config.Config{}
	cfg.Management.OpsAlertsEnabled = true

	if NewOpsAlerter(&stubNotifierAPI{}, cfg) != nil {
		t.Fatal("expected nil without recipient")
	}
}

func TestFormatSlotIDs_TruncatesLongLists(t *testing.T) {
	slots := make([]int16, 20)
	for i := range slots {
		slots[i] = int16(i)
	}
	got := formatSlotIDs(slots)
	if !strings.Contains(got, "+8 more") {
		t.Fatalf("expected truncation, got %q", got)
	}
}

func TestOpsAlerter_CriticalEventBroadcast(t *testing.T) {
	stub := &stubNotifierAPI{}
	cfg := testOpsAlerterConfig()

	alerter := NewOpsAlerter(stub, cfg)
	require.NotNil(t, alerter)

	err := alerter.enqueueNotification(context.Background(), "recon", "recon", "body", true)
	require.NoError(t, err)

	requests := stub.snapshot()
	require.Len(t, requests, 1)
	assert.True(t, requests[0].Broadcast)
	assert.Len(t, requests[0].BroadcastProviders, 3)
}

func TestOpsAlerter_WarningEventFallback(t *testing.T) {
	stub := &stubNotifierAPI{}
	cfg := testOpsAlerterConfig()

	alerter := NewOpsAlerter(stub, cfg)
	require.NotNil(t, alerter)

	err := alerter.enqueueNotification(context.Background(), "migration", "migration", "body", false)
	require.NoError(t, err)

	requests := stub.snapshot()
	require.Len(t, requests, 1)
	assert.False(t, requests[0].Broadcast)
}

func TestOpsAlerter_EnqueueFailureMetaAlert(t *testing.T) {
	stub := &stubNotifierAPI{fail: true}
	cfg := testOpsAlerterConfig()

	alerter := NewOpsAlerter(stub, cfg)
	require.NotNil(t, alerter)

	err := alerter.enqueueNotification(context.Background(), "test", "test", "body", false)
	require.Error(t, err)
}
