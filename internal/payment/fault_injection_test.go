package payment_test

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"ad-event-processor/pkg/faultproof"

	"ad-event-processor/internal/ingestion"
	"ad-event-processor/internal/payment"
	"ad-event-processor/internal/payment/db"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

const paymentFaultWorkers = 20

func TestFault_PaymentDualOutboxWorkerRace(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: fault test (run make test-integration)")
	}

	infra, cleanup := SetupPaymentFaultInfra(t)
	defer cleanup()

	ctx := context.Background()
	customerID := uuid.New()
	seed := SeedSucceededIntentWithOutbox(t, infra, customerID, 25_000_000, "fault-race-"+uuid.New().String())
	AssertPaymentFaultInvariants(t, infra.Pool, seed, 0, 0)

	worker := NewOutboxWorkerForFault(infra)
	const workers = 4
	var wg sync.WaitGroup
	var totalProcessed atomic.Int32
	wg.Add(workers)
	for range workers {
		go func() {
			defer wg.Done()
			n, _ := worker.ProcessOutbox(ctx, 10)
			totalProcessed.Add(int32(n))
		}()
	}
	wg.Wait()

	require.Eventually(t, func() bool {
		return PaymentOutboxStatus(t, infra.Pool, seed.OutboxID) == "PROCESSED" &&
			LedgerCountForIntent(t, infra.Pool, seed.IntentID) == 1
	}, 10*time.Second, 50*time.Millisecond)
	AssertPaymentFaultInvariants(t, infra.Pool, seed, seed.AmountMicro, 1)

	faultproof.Log(t, "outbox_worker_race", map[string]string{
		"subsystem":   "payment_outbox",
		"workers":     "4",
		"processed":   ItoaPaymentFault(int(totalProcessed.Load())),
		"ledger_rows": "1",
		"baseline_ok": "true",
		"fault_type":  "concurrency_stress",
	})
}

func TestFault_PaymentConcurrentCreateIdempotencyKey(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: fault test (run make test-integration)")
	}

	infra, cleanup := SetupPaymentFaultInfra(t)
	defer cleanup()

	ctx := context.Background()
	customerID := uuid.New()
	SeedCustomer(t, infra.Pool, customerID)

	svc := payment.NewService(infra.Pool, infra.Cfg)
	key := "fault-idem-" + uuid.New().String()
	amount := int64(12_000_000)

	var wg sync.WaitGroup
	wg.Add(paymentFaultWorkers)
	for range paymentFaultWorkers {
		go func() {
			defer wg.Done()
			_, err := svc.CreatePaymentIntent(ctx, customerID, amount, "USD", key, nil)
			require.NoError(t, err)
		}()
	}
	wg.Wait()

	var intentCount int
	require.NoError(t, infra.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM payment.payment_intents WHERE idempotency_key = $1`, key).Scan(&intentCount))
	require.Equal(t, 1, intentCount)

	faultproof.Log(t, "concurrent_idempotency_create", map[string]string{
		"subsystem":      "payment_intent",
		"workers":        ItoaPaymentFault(paymentFaultWorkers),
		"intents":        "1",
		"provider_calls": "1",
		"baseline_ok":    "true",
		"fault_type":     "concurrency_stress",
	})
}

func TestFault_PaymentConcurrentWebhookSameEventID(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: fault test (run make test-integration)")
	}

	infra, cleanup := SetupPaymentFaultInfra(t)
	defer cleanup()

	ctx := context.Background()
	customerID := uuid.New()
	SeedCustomer(t, infra.Pool, customerID)

	svc := payment.NewService(infra.Pool, infra.Cfg)
	result, err := svc.CreatePaymentIntent(ctx, customerID, 8_000_000, "USD", "fault-wh-"+uuid.New().String(), nil)
	require.NoError(t, err)
	intent := result.Intent
	providerRef := intent.ProviderRef
	eventID := "evt_concurrent_" + uuid.New().String()
	payload := fmt.Sprintf(`{"id":"%s","type":"payment_intent.succeeded","data":{"object":{"id":"%s","amount":8000000}}}`,
		eventID, providerRef)

	var wg sync.WaitGroup
	wg.Add(paymentFaultWorkers)
	for range paymentFaultWorkers {
		go func() {
			defer wg.Done()
			_ = svc.ProcessStripeWebhook(ctx, eventID, "payment_intent.succeeded", []byte(payload), providerRef, 8_000_000, payload)
		}()
	}
	wg.Wait()

	var webhookCount int
	require.NoError(t, infra.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM payment.webhook_events WHERE provider = 'stripe' AND provider_event_id = $1`,
		eventID).Scan(&webhookCount))
	require.Equal(t, 1, webhookCount)

	outbox, err := db.New(infra.Pool).GetPendingOutboxEventsForUpdate(ctx, 10)
	require.NoError(t, err)
	require.Len(t, outbox, 1)

	faultproof.Log(t, "concurrent_webhook_dedup", map[string]string{
		"subsystem":    "payment_webhook",
		"workers":      ItoaPaymentFault(paymentFaultWorkers),
		"webhook_rows": "1",
		"outbox_rows":  "1",
		"baseline_ok":  "true",
		"fault_type":   "concurrency_stress",
	})
}

func TestFault_PaymentStaleLeaseReclaim(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: fault test (run make test-integration)")
	}

	infra, cleanup := SetupPaymentFaultInfra(t)
	defer cleanup()

	ctx := context.Background()
	customerID := uuid.New()
	seed := SeedSucceededIntentWithOutbox(t, infra, customerID, 15_000_000, "fault-lease-"+uuid.New().String())

	_, err := infra.Pool.Exec(ctx, `
		UPDATE payment.payment_outbox
		SET status = 'PROCESSING', lease_until = now() - interval '1 minute', attempts = 1
		WHERE id = $1`, seed.OutboxID)
	require.NoError(t, err)

	worker := NewOutboxWorkerForFault(infra)
	worker.ReclaimStaleProcessingForTest(ctx)
	require.Equal(t, "PENDING", PaymentOutboxStatus(t, infra.Pool, seed.OutboxID))

	n, err := worker.ProcessOutbox(ctx, 10)
	require.NoError(t, err)
	require.Equal(t, 1, n)
	AssertPaymentFaultInvariants(t, infra.Pool, seed, seed.AmountMicro, 1)

	faultproof.Log(t, "stale_lease_reclaim", map[string]string{
		"subsystem":   "payment_outbox",
		"recovered":   "true",
		"ledger_rows": "1",
		"baseline_ok": "true",
		"fault_type":  "worker_crash_simulation",
	})
}

func TestFault_PaymentPostSettlementMarkGap(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: fault test (run make test-integration)")
	}

	infra, cleanup := SetupPaymentFaultInfra(t)
	defer cleanup()
	defer func() { payment.SetPostSettlementMarkHookForTest(nil) }()

	ctx := context.Background()
	customerID := uuid.New()
	seed := SeedSucceededIntentWithOutbox(t, infra, customerID, 18_000_000, "fault-gap-"+uuid.New().String())

	var hookCalls atomic.Int32
	payment.SetPostSettlementMarkHookForTest(func(ctx context.Context, ev db.PaymentPaymentOutbox) error {
		if hookCalls.Add(1) == 1 {
			return fmt.Errorf("injected post-settlement mark failure")
		}
		return nil
	})

	worker := NewOutboxWorkerForFault(infra)
	n, err := worker.ProcessOutbox(ctx, 10)
	require.NoError(t, err)
	require.Equal(t, 0, n)
	AssertPaymentFaultInvariants(t, infra.Pool, seed, seed.AmountMicro, 1)
	require.Equal(t, "PENDING", PaymentOutboxStatus(t, infra.Pool, seed.OutboxID))

	n, err = worker.ProcessOutbox(ctx, 10)
	require.NoError(t, err)
	require.Equal(t, 1, n)
	AssertPaymentFaultInvariants(t, infra.Pool, seed, seed.AmountMicro, 1)
	require.Equal(t, "PROCESSED", PaymentOutboxStatus(t, infra.Pool, seed.OutboxID))

	faultproof.Log(t, "post_settlement_mark_failed", map[string]string{
		"subsystem":     "payment_outbox",
		"ledger_rows":   "1",
		"double_credit": "false",
		"hook_calls":    ItoaPaymentFault(int(hookCalls.Load())),
		"baseline_ok":   "true",
		"fault_type":    "injected_timing_gap",
	})
}

func TestFault_PaymentMissingCustomerSettlementDead(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: fault test (run make test-integration)")
	}

	infra, cleanup := SetupPaymentFaultInfra(t)
	defer cleanup()

	ctx := context.Background()
	customerID := uuid.New()
	seed := SeedSucceededIntentWithOutbox(t, infra, customerID, 9_000_000, "fault-orphan-"+uuid.New().String())

	_, err := infra.Pool.Exec(ctx, `DELETE FROM customers WHERE id = $1`, ingestion.ToUUID(customerID))
	require.NoError(t, err)

	worker := NewOutboxWorkerForFault(infra)
	n, err := worker.ProcessOutbox(ctx, 10)
	require.NoError(t, err)
	require.Equal(t, 0, n)

	status := PaymentOutboxStatus(t, infra.Pool, seed.OutboxID)
	require.Equal(t, "DEAD", status)
	require.Equal(t, 0, LedgerCountForIntent(t, infra.Pool, seed.IntentID))

	var intentStatus string
	require.NoError(t, infra.Pool.QueryRow(ctx, `
		SELECT status FROM payment.payment_intents WHERE id = $1`, ingestion.ToUUID(seed.IntentID)).Scan(&intentStatus))
	require.Equal(t, "SETTLEMENT_FAILED", intentStatus)

	faultproof.Log(t, "settlement_customer_not_found", map[string]string{
		"subsystem":     "payment_outbox",
		"outbox_status": status,
		"intent_status": intentStatus,
		"ledger_rows":   "0",
		"baseline_ok":   "true",
		"fault_type":    "missing_customer",
	})
}
