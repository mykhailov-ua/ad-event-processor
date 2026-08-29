package controlplane

import (
	"context"
	"testing"
	"time"

	"ad-event-processor/internal/opsadmin"

	"ad-event-processor/pkg/faultproof"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpsAlerter_AlertOutboxStuck(t *testing.T) {
	stub := &stubNotifierAPITest{}
	cfg := testNotifierConfig()
	cfg.Management.OpsAlertsEnabled = true

	alerter := opsadmin.NewOpsAlerter(stub, cfg)
	require.NotNil(t, alerter)
	assert.Equal(t, 120, alerter.OutboxStuckThresholdSec())

	alerter.AlertOutboxStuck(context.Background(), 12, 180)
	time.Sleep(100 * time.Millisecond)

	requests := stub.snapshot()
	require.Len(t, requests, 1)
	assert.True(t, requests[0].Broadcast)
}

func TestOpsAlerter_AlertCHEmergencyDrop(t *testing.T) {
	stub := &stubNotifierAPITest{}
	cfg := testNotifierConfig()
	cfg.Management.OpsAlertsEnabled = true

	alerter := opsadmin.NewOpsAlerter(stub, cfg)
	require.NotNil(t, alerter)

	alerter.AlertCHEmergencyDrop(context.Background(), "impressions", "202401", 92.5, 90)
	time.Sleep(100 * time.Millisecond)

	requests := stub.snapshot()
	require.Len(t, requests, 1)
	assert.Contains(t, requests[0].Body, "CH emergency drop")
	assert.True(t, requests[0].Broadcast)
}

func TestOpsAlerter_AlertBlacklistJanitorFailed(t *testing.T) {
	stub := &stubNotifierAPITest{}
	cfg := testNotifierConfig()
	cfg.Management.OpsAlertsEnabled = true

	alerter := opsadmin.NewOpsAlerter(stub, cfg)
	require.NotNil(t, alerter)

	alerter.AlertBlacklistJanitorFailed(context.Background(), assert.AnError)
	time.Sleep(100 * time.Millisecond)

	requests := stub.snapshot()
	require.Len(t, requests, 1)
	assert.Contains(t, requests[0].Body, "Blacklist janitor")
}

func TestOpsAlerter_AlertSlotMigrationError(t *testing.T) {
	stub := &stubNotifierAPITest{}
	cfg := testNotifierConfig()
	cfg.Management.OpsAlertsEnabled = true

	alerter := opsadmin.NewOpsAlerter(stub, cfg)
	require.NotNil(t, alerter)

	alerter.AlertSlotMigrationError(context.Background(), "copy", assert.AnError)
	time.Sleep(100 * time.Millisecond)

	requests := stub.snapshot()
	require.Len(t, requests, 1)
	assert.Contains(t, requests[0].Title, "slot migration copy failed")
}

func TestOutboxMetrics_AlertsWhenStale(t *testing.T) {
	stub := &stubNotifierAPITest{}
	cfg := testNotifierConfig()
	cfg.Management.OpsAlertsEnabled = true
	cfg.Management.OpsAlertOutboxStuckSec = 60

	svc := &Service{alerter: opsadmin.NewOpsAlerter(stub, cfg)}
	worker := NewOutboxWorker(svc)

	worker.RecordOutboxLagFromValues(context.Background(), 5, 90)
	time.Sleep(100 * time.Millisecond)

	requests := stub.snapshot()
	require.Len(t, requests, 1)
	assert.Contains(t, requests[0].Body, "Outbox backlog stale")
}

func TestFault_opsAlertExtendedCoverage(t *testing.T) {
	stub := &stubNotifierAPITest{}
	cfg := testNotifierConfig()
	cfg.Management.OpsAlertsEnabled = true

	alerter := opsadmin.NewOpsAlerter(stub, cfg)
	require.NotNil(t, alerter)

	alerter.AlertBlacklistJanitorFailed(context.Background(), assert.AnError)
	alerter.AlertOutboxStuck(context.Background(), 3, 150)
	alerter.AlertSlotMigrationError(context.Background(), "drain", assert.AnError)
	time.Sleep(150 * time.Millisecond)

	requests := stub.snapshot()
	require.Len(t, requests, 3)
	for _, req := range requests {
		assert.True(t, req.Broadcast)
	}

	faultproof.Log(t, "ops_alert_extended", map[string]string{
		"blacklist": "true",
		"outbox":    "true",
		"migration": "true",
		"broadcast": "true",
	})
}
