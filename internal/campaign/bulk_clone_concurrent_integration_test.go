package campaign

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"ad-event-processor/internal/testutil"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestBulkClone_concurrentTen_noDeadlock(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: bulk clone advisory lock concurrency")
	}
	pool, cleanup := testutil.SetupAdsPostgres(t)
	defer cleanup()

	customerID := uuid.New()
	const workers = 10
	done := make(chan struct{}, workers)
	var okCount atomic.Int32

	for i := 0; i < workers; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			err := WithBulkCloneCustomerLock(context.Background(), pool, customerID, func() error {
				time.Sleep(5 * time.Millisecond)
				okCount.Add(1)
				return nil
			})
			require.NoError(t, err)
		}()
	}

	deadline := time.After(30 * time.Second)
	for i := 0; i < workers; i++ {
		select {
		case <-done:
		case <-deadline:
			t.Fatal("bulk clone lock workers did not finish within 30s")
		}
	}
	require.Equal(t, int32(workers), okCount.Load())
}
