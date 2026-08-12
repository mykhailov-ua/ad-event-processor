package notify

import (
	"github.com/bidshard/ad-event-processor/pkg/faultproof"

	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/bidshard/ad-event-processor/internal/notify/db"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFault_notifierConcurrentDelivery(t *testing.T) {
	if testing.Short() {
		t.Skip("notifier fault integration test")
	}

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	svc := newTestService(pool)
	ctx := context.Background()

	const notifications = 5
	for i := range notifications {
		_, err := sendTestNotification(ctx, svc, NotificationInput{
			Provider:  string(db.NotifierProviderTELEGRAM),
			Recipient: fmt.Sprintf("chat-%d", i),
			Title:     "Concurrent test",
			Body:      fmt.Sprintf("body %d", i),
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

	sentCount, err := countNotificationsByStatus(ctx, pool, db.NotifierNotificationStatusSENT)
	require.NoError(t, err)
	assert.Equal(t, notifications, sentCount)

	faultproof.Log(t, "notifier_concurrent_delivery", map[string]string{
		"workers":       fmt.Sprintf("%d", workerCount),
		"notifications": fmt.Sprintf("%d", notifications),
		"sent_total":    fmt.Sprintf("%d", sentCount),
		"double_send":   "false",
	})
}
