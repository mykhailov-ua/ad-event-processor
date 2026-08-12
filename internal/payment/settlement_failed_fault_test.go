package payment_test

import (
	"github.com/bidshard/ad-event-processor/pkg/faultproof"

	"context"
	"fmt"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/internal/ingestion"
	"github.com/bidshard/ad-event-processor/internal/payment"
	"github.com/bidshard/ad-event-processor/internal/payment/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func TestFault_SettlementFailedNotifier(t *testing.T) {
	if testing.Short() {
		t.Skip("fault integration test")
	}

	infra, cleanup := SetupPaymentFaultInfra(t)
	defer cleanup()

	stub := &StubNotifierAPI{}
	cfg := faultTestOpsConfig()
	alerter := payment.NewSettlementFailedAlerter(payment.NewInProcessNotifierClient(stub), cfg)
	require.NotNil(t, alerter)

	ctx := context.Background()
	customerID := uuid.New()
	seed := SeedSucceededIntentWithOutbox(t, infra, customerID, 9_000_000, "fault-settle-alert-"+uuid.New().String())

	_, err := infra.Pool.Exec(ctx, `DELETE FROM customers WHERE id = $1`, ingestion.ToUUID(customerID))
	require.NoError(t, err)

	worker := NewOutboxWorkerForFault(infra)
	worker.SetSettlementFailedAlerter(alerter)

	n, err := worker.ProcessOutbox(ctx, 10)
	require.NoError(t, err)
	require.Equal(t, 0, n)
	require.Equal(t, "DEAD", PaymentOutboxStatus(t, infra.Pool, seed.OutboxID))

	time.Sleep(200 * time.Millisecond)
	requests := stub.Snapshot()
	require.Len(t, requests, 1)
	require.Equal(t, "payment-settlement-failed:"+seed.IntentID.String(), requests[0].DedupKey)
	require.Contains(t, requests[0].Body, seed.IntentID.String())

	alerter.AlertPermanentFailure(
		loadPaymentOutboxRow(t, infra.Pool, seed.OutboxID),
		fmt.Errorf("customer not found"),
	)
	time.Sleep(100 * time.Millisecond)
	require.Len(t, stub.Snapshot(), 1, "cooldown should suppress duplicate alert for same intent")

	faultproof.Log(t, "settlement_failed_notifier", map[string]string{
		"subsystem":   "payment_outbox",
		"intent_id":   seed.IntentID.String(),
		"notified":    "true",
		"dedup_key":   requests[0].DedupKey,
		"baseline_ok": "true",
		"fault_type":  "missing_customer",
	})
}

func loadPaymentOutboxRow(t *testing.T, pool *pgxpool.Pool, outboxID int64) db.PaymentPaymentOutbox {
	t.Helper()
	var ev db.PaymentPaymentOutbox
	require.NoError(t, pool.QueryRow(context.Background(), `
		SELECT id, event_type, payload, status, lease_until, attempts, last_error, created_at, processed_at
		FROM payment.payment_outbox WHERE id = $1`, outboxID).Scan(
		&ev.ID, &ev.EventType, &ev.Payload, &ev.Status, &ev.LeaseUntil, &ev.Attempts, &ev.LastError, &ev.CreatedAt, &ev.ProcessedAt,
	))
	return ev
}
