package notifier

import (
	"espx/pkg/faultproof"

	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"espx/internal/notifier/pb"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newBroadcastMockProviders(failProvider pb.Provider) map[pb.Provider]Provider {
	slackBreaker := NewCircuitBreaker(10, 2, 10*time.Second)
	telegramBreaker := NewCircuitBreaker(10, 2, 10*time.Second)
	smsBreaker := NewCircuitBreaker(10, 2, 10*time.Second)

	mockSlack := NewMockProvider(slackBreaker)
	mockSlack.ProviderName = "SLACK"
	mockSlack.ShouldFail = failProvider == pb.Provider_PROVIDER_SLACK

	mockTelegram := NewMockProvider(telegramBreaker)
	mockTelegram.ProviderName = "TELEGRAM"
	mockTelegram.ShouldFail = failProvider == pb.Provider_PROVIDER_TELEGRAM

	mockSMS := NewMockProvider(smsBreaker)
	mockSMS.ProviderName = "SMS"
	mockSMS.ShouldFail = failProvider == pb.Provider_PROVIDER_SMS

	return map[pb.Provider]Provider{
		pb.Provider_PROVIDER_SLACK:    mockSlack,
		pb.Provider_PROVIDER_TELEGRAM: mockTelegram,
		pb.Provider_PROVIDER_SMS:      mockSMS,
	}
}

func TestFault_notifierBroadcastPartialFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("notifier fault integration test")
	}

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	providers := newBroadcastMockProviders(pb.Provider_PROVIDER_SLACK)
	svc := NewService(pool, providers)
	ctx := context.Background()

	resp, err := svc.SendNotification(ctx, &pb.SendNotificationRequest{
		Provider:     pb.Provider_PROVIDER_SLACK,
		Recipient:    "https://hooks.slack.com/services/test",
		Title:        "Critical incident",
		Body:         "broadcast partial failure probe",
		DeliveryMode: pb.DeliveryMode_DELIVERY_MODE_BROADCAST,
	})
	require.NoError(t, err)

	processed, err := svc.ProcessPending(ctx, workerBatchSize)
	require.NoError(t, err)
	assert.Equal(t, 1, processed)

	getResp, err := svc.GetNotification(ctx, &pb.GetNotificationRequest{NotificationId: resp.NotificationId})
	require.NoError(t, err)
	assert.Equal(t, pb.NotificationStatus_NOTIFICATION_STATUS_SENT, getResp.Notification.Status)
	assert.Contains(t, getResp.Notification.ErrorMessage, "broadcast partial")

	mockSlack := providers[pb.Provider_PROVIDER_SLACK].(*MockProvider)
	mockTelegram := providers[pb.Provider_PROVIDER_TELEGRAM].(*MockProvider)
	mockSMS := providers[pb.Provider_PROVIDER_SMS].(*MockProvider)

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

	providers := newBroadcastMockProviders(pb.Provider_PROVIDER_UNSPECIFIED)
	svc := NewService(pool, providers)
	ctx := context.Background()

	const notifications = 4
	for i := range notifications {
		_, err := svc.SendNotification(ctx, &pb.SendNotificationRequest{
			Provider:     pb.Provider_PROVIDER_TELEGRAM,
			Recipient:    fmt.Sprintf("chat-%d", i),
			Title:        "Broadcast concurrent",
			Body:         fmt.Sprintf("body %d", i),
			DeliveryMode: pb.DeliveryMode_DELIVERY_MODE_BROADCAST,
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

	mockSlack := providers[pb.Provider_PROVIDER_SLACK].(*MockProvider)
	mockTelegram := providers[pb.Provider_PROVIDER_TELEGRAM].(*MockProvider)
	mockSMS := providers[pb.Provider_PROVIDER_SMS].(*MockProvider)

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

	providers := map[pb.Provider]Provider{
		pb.Provider_PROVIDER_SLACK:    mockSlack,
		pb.Provider_PROVIDER_TELEGRAM: mockTelegram,
		pb.Provider_PROVIDER_SMS:      mockSMS,
	}

	svc := NewService(pool, providers)
	ctx := context.Background()

	resp, err := svc.SendNotification(ctx, &pb.SendNotificationRequest{
		Provider:     pb.Provider_PROVIDER_TELEGRAM,
		Recipient:    "chat-retry",
		Title:        "Broadcast retry",
		Body:         "all fail probe",
		DeliveryMode: pb.DeliveryMode_DELIVERY_MODE_BROADCAST,
	})
	require.NoError(t, err)

	processed, err := svc.ProcessPending(ctx, workerBatchSize)
	require.NoError(t, err)
	assert.Equal(t, 1, processed)

	getResp, err := svc.GetNotification(ctx, &pb.GetNotificationRequest{NotificationId: resp.NotificationId})
	require.NoError(t, err)
	assert.Equal(t, pb.NotificationStatus_NOTIFICATION_STATUS_PENDING, getResp.Notification.Status)
	assert.Equal(t, int32(1), getResp.Notification.RetryCount)

	processed, err = svc.ProcessPending(ctx, workerBatchSize)
	require.NoError(t, err)
	assert.Equal(t, 0, processed, "expected backoff to skip immediate retry")

	mockTelegram.ShouldFail = false
	mockSMS.ShouldFail = false

	id, err := pgUUIDFromString(resp.NotificationId)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, "UPDATE notifier.notifications SET updated_at = now() - interval '10 seconds' WHERE id = $1", id)
	require.NoError(t, err)

	processed, err = svc.ProcessPending(ctx, workerBatchSize)
	require.NoError(t, err)
	assert.Equal(t, 1, processed)

	getResp, err = svc.GetNotification(ctx, &pb.GetNotificationRequest{NotificationId: resp.NotificationId})
	require.NoError(t, err)
	assert.Equal(t, pb.NotificationStatus_NOTIFICATION_STATUS_SENT, getResp.Notification.Status)
	assert.Contains(t, getResp.Notification.ErrorMessage, "broadcast partial")

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

	providers := map[pb.Provider]Provider{
		pb.Provider_PROVIDER_SLACK:    &MockProvider{breaker: slackBreaker, ProviderName: "SLACK"},
		pb.Provider_PROVIDER_TELEGRAM: NewMockProvider(NewCircuitBreaker(10, 2, 10*time.Second)),
		pb.Provider_PROVIDER_SMS:      NewMockProvider(NewCircuitBreaker(10, 2, 10*time.Second)),
	}
	providers[pb.Provider_PROVIDER_TELEGRAM].(*MockProvider).ProviderName = "TELEGRAM"
	providers[pb.Provider_PROVIDER_SMS].(*MockProvider).ProviderName = "SMS"

	svc := NewService(pool, providers)
	ctx := context.Background()

	resp, err := svc.SendNotification(ctx, &pb.SendNotificationRequest{
		Provider:     pb.Provider_PROVIDER_SLACK,
		Recipient:    "https://hooks.slack.com/services/test",
		Title:        "Circuit open probe",
		Body:         "broadcast with open circuit",
		DeliveryMode: pb.DeliveryMode_DELIVERY_MODE_BROADCAST,
	})
	require.NoError(t, err)

	processed, err := svc.ProcessPending(ctx, workerBatchSize)
	require.NoError(t, err)
	assert.Equal(t, 1, processed)

	getResp, err := svc.GetNotification(ctx, &pb.GetNotificationRequest{NotificationId: resp.NotificationId})
	require.NoError(t, err)
	assert.Equal(t, pb.NotificationStatus_NOTIFICATION_STATUS_SENT, getResp.Notification.Status)

	mockSlack := providers[pb.Provider_PROVIDER_SLACK].(*MockProvider)
	mockTelegram := providers[pb.Provider_PROVIDER_TELEGRAM].(*MockProvider)
	assert.Len(t, mockSlack.Sent, 0)
	assert.Len(t, mockTelegram.Sent, 1)

	faultproof.Log(t, "notifier_broadcast_circuit_open", map[string]string{
		"open_provider": "SLACK",
		"quorum":        "true",
		"status":        "SENT",
	})
}
