package payment_test

import (
	"espx/pkg/faultproof"

	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"espx/internal/ingestion"
	"espx/internal/payment"
	"espx/internal/payment/db"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestFault_PaymentDisputeConcurrentWebhookSameEventID(t *testing.T) {
	if testing.Short() {
		t.Skip("fault integration test")
	}

	infra, cleanup := SetupPaymentFaultInfra(t)
	defer cleanup()

	ctx := context.Background()
	customerID := uuid.New()
	seed := SeedSettledIntent(t, infra, customerID, 18_000_000, "fault-dp-wh-"+uuid.New().String())
	svc := payment.NewService(infra.Pool, infra.Cfg)

	eventID := "evt_dispute_concurrent_" + uuid.New().String()
	disputeID := "dp_" + uuid.New().String()

	var wg sync.WaitGroup
	wg.Add(paymentFaultWorkers)
	for i := 0; i < paymentFaultWorkers; i++ {
		go func() {
			defer wg.Done()
			stripeCents, _ := payment.MicroToStripeAmount(10_000_000)
			payload := fmt.Sprintf(`{"id":"%s","type":"charge.dispute.funds_withdrawn","data":{"object":{"id":"%s","amount":%d,"payment_intent":"%s","status":"needs_response"}}}`,
				eventID, disputeID, stripeCents, seed.ProviderRef)
			_ = svc.ProcessStripeDisputeWebhook(ctx, eventID, "charge.dispute.funds_withdrawn", []byte(payload), disputeID, seed.ProviderRef, 10_000_000, "needs_response")
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
	require.Equal(t, payment.OutboxEventApplyChargeback, outbox[0].EventType)

	faultproof.Log(t, "concurrent_dispute_webhook_dedup", map[string]string{
		"subsystem":    "payment_dispute_webhook",
		"workers":      ItoaPaymentFault(paymentFaultWorkers),
		"webhook_rows": "1",
		"outbox_rows":  "1",
		"baseline_ok":  "true",
		"fault_type":   "concurrency_stress",
	})
}

func TestFault_PaymentChargebackDualOutboxWorkerRace(t *testing.T) {
	if testing.Short() {
		t.Skip("fault integration test")
	}

	infra, cleanup := SetupPaymentFaultInfra(t)
	defer cleanup()

	ctx := context.Background()
	customerID := uuid.New()
	seed := SeedSettledIntent(t, infra, customerID, 26_000_000, "fault-cb-race-"+uuid.New().String())
	svc := payment.NewService(infra.Pool, infra.Cfg)
	disputeID := "dp_race_" + uuid.New().String()
	ProcessDisputeWebhook(t, infra.Pool, svc, "evt_cb_race_"+uuid.New().String(), "charge.dispute.funds_withdrawn", seed.ProviderRef, disputeID, 11_000_000, "needs_response")
	outboxID := LatestOutboxIDByType(t, infra.Pool, payment.OutboxEventApplyChargeback)

	worker := NewOutboxWorkerForFault(infra)
	const workers = 4
	var wg sync.WaitGroup
	var totalProcessed atomic.Int32
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			n, _ := worker.ProcessOutbox(ctx, 10)
			totalProcessed.Add(int32(n))
		}()
	}
	wg.Wait()

	wantBalance := seed.AmountMicro - 11_000_000
	require.Eventually(t, func() bool {
		return PaymentOutboxStatus(t, infra.Pool, outboxID) == "PROCESSED" &&
			LedgerChargebackCountForIntent(t, infra.Pool, seed.IntentID) == 1
	}, 10*time.Second, 50*time.Millisecond)
	AssertPaymentChargebackInvariants(t, infra.Pool, seed, wantBalance, 1, 0)

	faultproof.Log(t, "chargeback_outbox_worker_race", map[string]string{
		"subsystem":   "payment_chargeback_outbox",
		"workers":     "4",
		"ledger_rows": "1",
		"baseline_ok": "true",
		"fault_type":  "concurrency_stress",
	})
}

func TestFault_PaymentChargebackPostSettlementMarkGap(t *testing.T) {
	if testing.Short() {
		t.Skip("fault integration test")
	}

	infra, cleanup := SetupPaymentFaultInfra(t)
	defer cleanup()
	defer func() { payment.SetPostSettlementMarkHookForTest(nil) }()

	ctx := context.Background()
	customerID := uuid.New()
	seed := SeedSettledIntent(t, infra, customerID, 28_000_000, "fault-cb-gap-"+uuid.New().String())
	svc := payment.NewService(infra.Pool, infra.Cfg)
	disputeID := "dp_gap_" + uuid.New().String()
	ProcessDisputeWebhook(t, infra.Pool, svc, "evt_cb_gap_"+uuid.New().String(), "charge.dispute.funds_withdrawn", seed.ProviderRef, disputeID, 14_000_000, "under_review")
	outboxID := LatestOutboxIDByType(t, infra.Pool, payment.OutboxEventApplyChargeback)

	var hookCalls atomic.Int32
	payment.SetPostSettlementMarkHookForTest(func(ctx context.Context, ev db.PaymentPaymentOutbox) error {
		if ev.EventType == payment.OutboxEventApplyChargeback && hookCalls.Add(1) == 1 {
			return fmt.Errorf("injected post-chargeback mark failure")
		}
		return nil
	})

	worker := NewOutboxWorkerForFault(infra)
	wantBalance := seed.AmountMicro - 14_000_000

	n, err := worker.ProcessOutbox(ctx, 10)
	require.NoError(t, err)
	require.Equal(t, 0, n)
	AssertPaymentChargebackInvariants(t, infra.Pool, seed, wantBalance, 1, 0)
	require.Equal(t, "PENDING", PaymentOutboxStatus(t, infra.Pool, outboxID))

	n, err = worker.ProcessOutbox(ctx, 10)
	require.NoError(t, err)
	require.Equal(t, 1, n)
	AssertPaymentChargebackInvariants(t, infra.Pool, seed, wantBalance, 1, 0)
	require.Equal(t, "PROCESSED", PaymentOutboxStatus(t, infra.Pool, outboxID))

	faultproof.Log(t, "post_chargeback_mark_failed", map[string]string{
		"subsystem":    "payment_chargeback_outbox",
		"ledger_rows":  "1",
		"double_debit": "false",
		"hook_calls":   ItoaPaymentFault(int(hookCalls.Load())),
		"baseline_ok":  "true",
		"fault_type":   "injected_timing_gap",
	})
}

func TestFault_PaymentDisputeWithdrawnThenReinstated(t *testing.T) {
	if testing.Short() {
		t.Skip("fault integration test")
	}

	infra, cleanup := SetupPaymentFaultInfra(t)
	defer cleanup()

	ctx := context.Background()
	customerID := uuid.New()
	seed := SeedSettledIntent(t, infra, customerID, 32_000_000, "fault-dp-cycle-"+uuid.New().String())
	svc := payment.NewService(infra.Pool, infra.Cfg)
	worker := NewOutboxWorkerForFault(infra)
	disputeID := "dp_cycle_" + uuid.New().String()

	ProcessDisputeWebhook(t, infra.Pool, svc, "evt_dp_created_"+uuid.New().String(), "charge.dispute.created", seed.ProviderRef, disputeID, 32_000_000, "needs_response")
	var intentStatus string
	require.NoError(t, infra.Pool.QueryRow(ctx, `
		SELECT status FROM payment.payment_intents WHERE id = $1`, ingestion.ToUUID(seed.IntentID)).Scan(&intentStatus))
	require.Equal(t, "DISPUTED", intentStatus)

	ProcessDisputeWebhook(t, infra.Pool, svc, "evt_dp_withdrawn_"+uuid.New().String(), "charge.dispute.funds_withdrawn", seed.ProviderRef, disputeID, 32_000_000, "needs_response")
	withdrawnOutbox := LatestOutboxIDByType(t, infra.Pool, payment.OutboxEventApplyChargeback)
	n, err := worker.ProcessOutbox(ctx, 10)
	require.NoError(t, err)
	require.Equal(t, 1, n)
	require.Equal(t, "PROCESSED", PaymentOutboxStatus(t, infra.Pool, withdrawnOutbox))
	AssertPaymentChargebackInvariants(t, infra.Pool, seed, 0, 1, 0)

	ProcessDisputeWebhook(t, infra.Pool, svc, "evt_dp_reinstated_"+uuid.New().String(), "charge.dispute.funds_reinstated", seed.ProviderRef, disputeID, 32_000_000, "won")
	reinstatedOutbox := LatestOutboxIDByType(t, infra.Pool, payment.OutboxEventReverseChargeback)
	n, err = worker.ProcessOutbox(ctx, 10)
	require.NoError(t, err)
	require.Equal(t, 1, n)
	require.Equal(t, "PROCESSED", PaymentOutboxStatus(t, infra.Pool, reinstatedOutbox))
	AssertPaymentChargebackInvariants(t, infra.Pool, seed, seed.AmountMicro, 1, 1)

	ProcessDisputeWebhook(t, infra.Pool, svc, "evt_dp_closed_"+uuid.New().String(), "charge.dispute.closed", seed.ProviderRef, disputeID, 32_000_000, "won")
	require.NoError(t, infra.Pool.QueryRow(ctx, `
		SELECT status FROM payment.payment_intents WHERE id = $1`, ingestion.ToUUID(seed.IntentID)).Scan(&intentStatus))
	require.Equal(t, "SUCCEEDED", intentStatus)

	faultproof.Log(t, "dispute_withdrawn_then_reinstated", map[string]string{
		"subsystem":     "payment_dispute",
		"intent_status": intentStatus,
		"balance":       ItoaPaymentFault(int(seed.AmountMicro / 1_000_000)),
		"baseline_ok":   "true",
		"fault_type":    "dispute_lifecycle",
	})
}

func TestFault_PaymentChargebackExceedsIntentIgnored(t *testing.T) {
	if testing.Short() {
		t.Skip("fault integration test")
	}

	infra, cleanup := SetupPaymentFaultInfra(t)
	defer cleanup()

	customerID := uuid.New()
	seed := SeedSettledIntent(t, infra, customerID, 10_000_000, "fault-cb-exceed-"+uuid.New().String())
	svc := payment.NewService(infra.Pool, infra.Cfg)

	ProcessDisputeWebhook(t, infra.Pool, svc, "evt_cb_exceed_"+uuid.New().String(), "charge.dispute.funds_withdrawn", seed.ProviderRef, "dp_exceed", 15_000_000, "needs_response")

	outbox, err := db.New(infra.Pool).GetPendingOutboxEventsForUpdate(context.Background(), 10)
	require.NoError(t, err)
	require.Empty(t, outbox)
	AssertPaymentFaultInvariants(t, infra.Pool, seed, seed.AmountMicro, 1)

	faultproof.Log(t, "chargeback_exceeds_intent_ignored", map[string]string{
		"subsystem":   "payment_dispute_webhook",
		"outbox_rows": "0",
		"baseline_ok": "true",
		"fault_type":  "invalid_chargeback_amount",
	})
}

func TestFault_PaymentChargebackWithoutTopupDead(t *testing.T) {
	if testing.Short() {
		t.Skip("fault integration test")
	}

	infra, cleanup := SetupPaymentFaultInfra(t)
	defer cleanup()

	ctx := context.Background()
	customerID := uuid.New()
	seed := SeedSucceededIntentWithOutbox(t, infra, customerID, 9_000_000, "fault-cb-no-topup-"+uuid.New().String())
	_, err := infra.Pool.Exec(ctx, `DELETE FROM payment.payment_outbox WHERE event_type = 'SETTLE_BALANCE'`)
	require.NoError(t, err)

	svc := payment.NewService(infra.Pool, infra.Cfg)
	ProcessDisputeWebhook(t, infra.Pool, svc, "evt_cb_no_topup_"+uuid.New().String(), "charge.dispute.funds_withdrawn", seed.ProviderRef, "dp_no_topup", 9_000_000, "needs_response")
	outboxID := LatestOutboxIDByType(t, infra.Pool, payment.OutboxEventApplyChargeback)

	worker := NewOutboxWorkerForFault(infra)
	n, err := worker.ProcessOutbox(ctx, 10)
	require.NoError(t, err)
	require.Equal(t, 0, n)
	require.Equal(t, "DEAD", PaymentOutboxStatus(t, infra.Pool, outboxID))
	require.Equal(t, 0, LedgerChargebackCountForIntent(t, infra.Pool, seed.IntentID))
	AssertPaymentFaultInvariants(t, infra.Pool, seed, 0, 0)

	faultproof.Log(t, "chargeback_without_topup_dead", map[string]string{
		"subsystem":     "payment_chargeback_outbox",
		"outbox_status": "DEAD",
		"ledger_rows":   "0",
		"baseline_ok":   "true",
		"fault_type":    "missing_topup",
	})
}
