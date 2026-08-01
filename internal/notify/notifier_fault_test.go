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

func TestFault_notifierConcurrentDelivery(t *testing.T) {
	if testing.Short() {
		t.Skip("notifier fault integration test")
	}

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	breaker := NewCircuitBreaker(100, 2, 10*time.Second)
	mockProv := NewMockProvider(breaker)
	providers := map[db.NotifierProvider]Provider{
		db.NotifierProviderTELEGRAM: mockProv,
	}
	svc := NewService(pool, providers)
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
	assert.Len(t, mockProv.Sent, notifications)

	faultproof.Log(t, "notifier_concurrent_delivery", map[string]string{
		"workers":       fmt.Sprintf("%d", workerCount),
		"notifications": fmt.Sprintf("%d", notifications),
		"sent_total":    fmt.Sprintf("%d", len(mockProv.Sent)),
		"double_send":   "false",
	})
}
