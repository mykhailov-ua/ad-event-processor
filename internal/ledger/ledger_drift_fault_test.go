package ledger

import (
	"context"
	"testing"
	"time"

	"ad-event-processor/pkg/faultproof"

	"github.com/stretchr/testify/require"
)

func TestFault_LedgerDriftCheck(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: fault test (run make test-integration)")
	}

	pool, cleanup := setupBillingTestDB(t)
	defer cleanup()

	ctx := context.Background()
	customerID := seedCustomerWithLedger(t, pool, time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC))

	_, err := pool.Exec(ctx, `UPDATE customers SET balance = balance + 100 WHERE id = $1`, customerID)
	require.NoError(t, err)

	svc := NewService(pool)
	month := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	_, err = svc.GenerateInvoice(ctx, customerID, month)
	require.ErrorIs(t, err, ErrLedgerDrift)

	faultproof.Log(t, "ledger_drift_check", map[string]string{
		"subsystem":    "billing",
		"customer_id":  customerID.String(),
		"rejected":     "true",
		"invariant_ok": "false",
	})
}
