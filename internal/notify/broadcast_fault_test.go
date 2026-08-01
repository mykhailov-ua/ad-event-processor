package notify

import (
	"espx/pkg/faultproof"

	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"espx/internal/notify/db"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newBroadcastMockProviders(failProvider db.NotifierProvider) map[db.NotifierProvider]Provider {
	slackBreaker := NewCircuitBreaker(10, 2, 10*time.Second)
	telegramBreaker := NewCircuitBreaker(10, 2, 10*time.Second)
	smsBreaker := NewCircuitBreaker(10, 2, 10*time.Second)

	mockSlack := NewMockProvider(slackBreaker)
	mockSlack.ProviderName = "SLACK"
	mockSlack.ShouldFail = failProvider == db.NotifierProviderSLACK

	mockTelegram := NewMockProvider(telegramBreaker)
	mockTelegram.ProviderName = "TELEGRAM"
	mockTelegram.ShouldFail = failProvider == db.NotifierProviderTELEGRAM

	mockSMS := NewMockProvider(smsBreaker)
	mockSMS.ProviderName = "SMS"
	mockSMS.ShouldFail = failProvider == db.NotifierProviderSMS

	return map[db.NotifierProvider]Provider{
		db.NotifierProviderSLACK:    mockSlack,
		db.NotifierProviderTELEGRAM: mockTelegram,
		db.NotifierProviderSMS:      mockSMS,
	}
}

func TestFault_notifierBroadcastPartialFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("notifier fault integration test")
	}

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	providers := newBroadcastMockProviders(db.NotifierProviderSLACK)
	svc := NewService(pool, providers)
	ctx := context.Background()

	result, err := sendTestNotification(ctx, svc, NotificationInput{
		Provider:  string(db.NotifierProviderSLACK),
		Recipient: "https://hooks.slack.com/services/test",
		Title:     "Critical incident",
		Body:      "broadcast partial failure probe",
		Broadcast: true,
	})
	require.NoError(t, err)

	processed, err := svc.ProcessPending(ctx, workerBatchSize)
	require.NoError(t, err)
	assert.Equal(t, 1, processed)

	notification, err := getTestNotification(ctx, svc, result.NotificationID)
	require.NoError(t, err)
	assert.Equal(t, db.NotifierNotificationStatusSENT, notification.Status)
	assert.Contains(t, notification.ErrorMessage, "broadcast partial")

	mockSlack := providers[db.NotifierProviderSLACK].(*MockProvider)
	mockTelegram := providers[db.NotifierProviderTELEGRAM].(*MockProvider)
	mockSMS := providers[db.NotifierProviderSMS].(*MockProvider)

	assert.Len(t, mockSlack.Sent, 0)
	assert.Len(t, mockTelegram.Sent, 1)
	assert.Len(t, mockSMS.Sent, 1)

	faultproof.Log(t, "notifier_broadcast_partial_failure", map[string]string{
		"failed_provider": "SLACK",
		"sent_total":      "2",
		"status":          "SENT",
		"quorum":          "true",
	})
}

func TestFault_notifierBroadcastConcurrentDelivery(t *testing.T) {
	if testing.Short() {
		t.Skip("notifier fault integration test")
	}

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	providers := newBroadcastMockProviders("")
	svc := NewService(pool, providers)
	ctx := context.Background()

	const notifications = 4
	for i := range notifications {
		_, err := sendTestNotification(ctx, svc, NotificationInput{
			Provider:  string(db.NotifierProviderTELEGRAM),
			Recipient: fmt.Sprintf("chat-%d", i),
			Title:     "Broadcast concurrent",
			Body:      fmt.Sprintf("body %d", i),
			Broadcast: true,
		})
		require.NoError(t, err)
	}

	var (
		wg          sync.WaitGroup
		processed   atomic.Int32
		errs        atomic.Int32
		workerCount = 24
	)

	for range workerCount {
		wg.Add(1)
		go func() {
			defer wg.Done()
			n, err := svc.ProcessPending(ctx, workerBatchSize)
			if err != nil {
				errs.Add(1)
				return
			}
			processed.Add(int32(n))
		}()
	}
	wg.Wait()

	assert.Equal(t, int32(0), errs.Load())
	assert.Equal(t, int32(notifications), processed.Load())

	mockSlack := providers[db.NotifierProviderSLACK].(*MockProvider)
	mockTelegram := providers[db.NotifierProviderTELEGRAM].(*MockProvider)
	mockSMS := providers[db.NotifierProviderSMS].(*MockProvider)

	assert.Len(t, mockSlack.Sent, notifications)
	assert.Len(t, mockTelegram.Sent, notifications)
	assert.Len(t, mockSMS.Sent, notifications)

	faultproof.Log(t, "notifier_broadcast_concurrent_delivery", map[string]string{
		"workers":       fmt.Sprintf("%d", workerCount),
		"notifications": fmt.Sprintf("%d", notifications),
		"channels":      "3",
		"double_send":   "false",
	})
}

func TestFault_notifierBroadcastAllFailThenRetry(t *testing.T) {
	if testing.Short() {
		t.Skip("notifier fault integration test")
	}

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	slackBreaker := NewCircuitBreaker(10, 2, 10*time.Second)
	telegramBreaker := NewCircuitBreaker(10, 2, 10*time.Second)
	smsBreaker := NewCircuitBreaker(10, 2, 10*time.Second)

	mockSlack := NewMockProvider(slackBreaker)
	mockSlack.ProviderName = "SLACK"
	mockSlack.ShouldFail = true
	mockTelegram := NewMockProvider(telegramBreaker)
	mockTelegram.ProviderName = "TELEGRAM"
	mockTelegram.ShouldFail = true
	mockSMS := NewMockProvider(smsBreaker)
	mockSMS.ProviderName = "SMS"
	mockSMS.ShouldFail = true

	providers := map[db.NotifierProvider]Provider{
		db.NotifierProviderSLACK:    mockSlack,
		db.NotifierProviderTELEGRAM: mockTelegram,
		db.NotifierProviderSMS:      mockSMS,
	}

	svc := NewService(pool, providers)
	ctx := context.Background()

	result, err := sendTestNotification(ctx, svc, NotificationInput{
		Provider:  string(db.NotifierProviderTELEGRAM),
		Recipient: "chat-retry",
		Title:     "Broadcast retry",
		Body:      "all fail probe",
		Broadcast: true,
	})
	require.NoError(t, err)

	processed, err := svc.ProcessPending(ctx, workerBatchSize)
	require.NoError(t, err)
	assert.Equal(t, 1, processed)

	notification, err := getTestNotification(ctx, svc, result.NotificationID)
	require.NoError(t, err)
	assert.Equal(t, db.NotifierNotificationStatusPENDING, notification.Status)
	assert.Equal(t, int32(1), notification.RetryCount)

	processed, err = svc.ProcessPending(ctx, workerBatchSize)
	require.NoError(t, err)
	assert.Equal(t, 0, processed, "expected backoff to skip immediate retry")

	mockTelegram.ShouldFail = false
	mockSMS.ShouldFail = false

	id, err := pgUUIDFromString(result.NotificationID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, "UPDATE notify.notifications SET updated_at = now() - interval '10 seconds' WHERE id = $1", id)
	require.NoError(t, err)

	processed, err = svc.ProcessPending(ctx, workerBatchSize)
	require.NoError(t, err)
	assert.Equal(t, 1, processed)

	notification, err = getTestNotification(ctx, svc, result.NotificationID)
	require.NoError(t, err)
	assert.Equal(t, db.NotifierNotificationStatusSENT, notification.Status)
	assert.Contains(t, notification.ErrorMessage, "broadcast partial")

	faultproof.Log(t, "notifier_broadcast_all_fail_retry", map[string]string{
		"initial_status": "PENDING",
		"final_status":   "SENT",
		"recovery":       "true",
	})
}

func TestFault_notifierBroadcastCircuitOpen(t *testing.T) {
	if testing.Short() {
		t.Skip("notifier fault integration test")
	}

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	slackBreaker := NewCircuitBreaker(2, 2, 10*time.Second)
	slackBreaker.trip()

	providers := map[db.NotifierProvider]Provider{
		db.NotifierProviderSLACK:    &MockProvider{breaker: slackBreaker, ProviderName: "SLACK"},
		db.NotifierProviderTELEGRAM: NewMockProvider(NewCircuitBreaker(10, 2, 10*time.Second)),
		db.NotifierProviderSMS:      NewMockProvider(NewCircuitBreaker(10, 2, 10*time.Second)),
	}
	providers[db.NotifierProviderTELEGRAM].(*MockProvider).ProviderName = "TELEGRAM"
	providers[db.NotifierProviderSMS].(*MockProvider).ProviderName = "SMS"

	svc := NewService(pool, providers)
	ctx := context.Background()

	result, err := sendTestNotification(ctx, svc, NotificationInput{
		Provider:  string(db.NotifierProviderSLACK),
		Recipient: "https://hooks.slack.com/services/test",
		Title:     "Circuit open probe",
		Body:      "broadcast with open circuit",
		Broadcast: true,
	})
	require.NoError(t, err)

	processed, err := svc.ProcessPending(ctx, workerBatchSize)
	require.NoError(t, err)
	assert.Equal(t, 1, processed)

	notification, err := getTestNotification(ctx, svc, result.NotificationID)
	require.NoError(t, err)
	assert.Equal(t, db.NotifierNotificationStatusSENT, notification.Status)

	mockSlack := providers[db.NotifierProviderSLACK].(*MockProvider)
	mockTelegram := providers[db.NotifierProviderTELEGRAM].(*MockProvider)
	assert.Len(t, mockSlack.Sent, 0)
	assert.Len(t, mockTelegram.Sent, 1)

	faultproof.Log(t, "notifier_broadcast_circuit_open", map[string]string{
		"open_provider": "SLACK",
		"quorum":        "true",
		"status":        "SENT",
	})
}
