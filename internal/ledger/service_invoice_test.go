package ledger

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_GenerateInvoice(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	pool, cleanup := setupBillingTestDB(t)
	defer cleanup()

	ctx := context.Background()
	month := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	feeAt := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	customerID := seedCustomerWithLedger(t, pool, feeAt)
	svc := NewService(pool)

	invoice, err := svc.GenerateInvoice(ctx, customerID, month)
	require.NoError(t, err)
	assert.NotEmpty(t, invoice.ID)
	assert.Equal(t, customerID.String(), invoice.CustomerID)
	assert.Equal(t, int64(2_500_000), invoice.SubtotalMicro)
	assert.Greater(t, invoice.TaxMicro, int64(0))
	assert.Equal(t, invoice.SubtotalMicro+invoice.TaxMicro, invoice.TotalMicro)

	again, err := svc.GenerateInvoice(ctx, customerID, month)
	require.NoError(t, err)
	assert.Equal(t, invoice.ID, again.ID)
}

func TestService_GenerateInvoiceConcurrent(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	pool, cleanup := setupBillingTestDB(t)
	defer cleanup()

	ctx := context.Background()
	month := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	feeAt := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	customerID := seedCustomerWithLedger(t, pool, feeAt)
	svc := NewService(pool)

	const goroutines = 20
	var wg sync.WaitGroup
	var success atomic.Int32
	ids := make(chan string, goroutines)

	wg.Add(goroutines)
	start := make(chan struct{})
	for range goroutines {
		go func() {
			defer wg.Done()
			<-start
			inv, err := svc.GenerateInvoice(ctx, customerID, month)
			if err == nil {
				success.Add(1)
				ids <- inv.ID
			}
		}()
	}
	close(start)
	wg.Wait()
	close(ids)

	assert.Equal(t, int32(goroutines), success.Load())
	first := ""
	for id := range ids {
		if first == "" {
			first = id
			continue
		}
		assert.Equal(t, first, id)
	}
}
