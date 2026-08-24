package ledger

import (
	"context"
	"testing"
	"time"

	"ad-event-processor/pkg/faultproof"

	"github.com/stretchr/testify/require"
)

func TestFault_InvoiceCronIdempotent(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: fault test (run make test-integration)")
	}

	pool, cleanup := setupBillingTestDB(t)
	defer cleanup()

	ctx := context.Background()
	month := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	feeAt := time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)
	customerID := seedCustomerWithLedger(t, pool, feeAt)
	svc := NewService(pool)
	worker := NewInvoiceWorker(svc)

	worker.RunInvoiceMonthForTest(ctx, month)
	first, err := svc.GenerateInvoice(ctx, customerID, month)
	require.NoError(t, err)

	worker.RunInvoiceMonthForTest(ctx, month)
	second, err := svc.GenerateInvoice(ctx, customerID, month)
	require.NoError(t, err)
	require.Equal(t, first.ID, second.ID)

	faultproof.Log(t, "invoice_cron_idempotent", map[string]string{
		"subsystem":   "billing",
		"customer_id": customerID.String(),
		"invoice_id":  first.ID,
		"idempotent":  "true",
	})
}
