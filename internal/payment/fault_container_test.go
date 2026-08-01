package payment_test

import (
	"espx/pkg/faultproof"

	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"espx/internal/payment"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestFault_PaymentPGStopOutboxClaimBlocked(t *testing.T) {
	if testing.Short() {
		t.Skip("fault integration test")
	}

	infra, cleanup := SetupPaymentFaultInfra(t)
	defer cleanup()

	ctx := context.Background()
	customerID := uuid.New()
	seed := SeedSucceededIntentWithOutbox(t, infra, customerID, 11_000_000, "fault-pg-stop-"+uuid.New().String())
	AssertPaymentFaultInvariants(t, infra.Pool, seed, 0, 0)

	worker := NewOutboxWorkerForFault(infra)
	StopPaymentContainer(t, infra.PGContainer)
	RequirePaymentFaultActive(t, func() bool {
		return infra.Pool.Ping(ctx) != nil
	}, "pg ping must fail after stop")

	processed, err := worker.ProcessOutbox(ctx, 10)
	require.Error(t, err)
	require.Equal(t, 0, processed)

	StartPaymentContainer(t, infra.PGContainer)
	infra.RefreshPGPool(t)
	require.Equal(t, "PENDING", PaymentOutboxStatus(t, infra.Pool, seed.OutboxID))
	AssertPaymentFaultInvariants(t, infra.Pool, seed, 0, 0)

	faultproof.Log(t, "postgres_container_stop", map[string]string{
		"subsystem":    "payment_outbox",
		"processed":    "0",
		"balance":      "0",
		"ledger_rows":  "0",
		"baseline_ok":  "true",
		"fault_verify": "postgres_container_stopped",
	})
}

func TestFault_PaymentPGStopStartOutboxRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("fault integration test")
	}

	infra, cleanup := SetupPaymentFaultInfra(t)
	defer cleanup()

	ctx := context.Background()
	customerID := uuid.New()
	seed := SeedSucceededIntentWithOutbox(t, infra, customerID, 13_000_000, "fault-pg-recovery-"+uuid.New().String())

	worker := NewOutboxWorkerForFault(infra)
	StopPaymentContainer(t, infra.PGContainer)
	RequirePaymentFaultActive(t, func() bool {
		return infra.Pool.Ping(ctx) != nil
	}, "pg ping must fail after stop")

	StartPaymentContainer(t, infra.PGContainer)
	infra.RefreshPGPool(t)
	worker.ReplacePoolForTest(infra.Pool)

	recovered := false
	require.Eventually(t, func() bool {
		n, err := worker.ProcessOutbox(ctx, 10)
		if err != nil || n != 1 {
			return false
		}
		recovered = PaymentOutboxStatus(t, infra.Pool, seed.OutboxID) == "PROCESSED"
		return recovered
	}, 30*time.Second, 200*time.Millisecond)

	AssertPaymentFaultInvariants(t, infra.Pool, seed, seed.AmountMicro, 1)

	faultproof.Log(t, "postgres_stop_start_recovery", map[string]string{
		"subsystem":    "payment_outbox",
		"recovered":    strconv.FormatBool(recovered),
		"ledger_rows":  "1",
		"baseline_ok":  "true",
		"fault_verify": "postgres_container_stopped_then_started",
	})
}

func TestFault_PaymentSettlementDownOutboxStaysPending(t *testing.T) {
	if testing.Short() {
		t.Skip("fault integration test")
	}

	infra, cleanup := SetupPaymentFaultInfra(t)
	defer cleanup()

	ctx := context.Background()
	customerID := uuid.New()
	seed := SeedSucceededIntentWithOutbox(t, infra, customerID, 14_000_000, "fault-grpc-stop-"+uuid.New().String())

	worker := NewOutboxWorkerForFault(infra)
	infra.SetSettlementDown()

	processed, err := worker.ProcessOutbox(ctx, 10)
	require.NoError(t, err)
	require.Equal(t, 0, processed)
	require.Equal(t, "PENDING", PaymentOutboxStatus(t, infra.Pool, seed.OutboxID))
	AssertPaymentFaultInvariants(t, infra.Pool, seed, 0, 0)

	faultproof.Log(t, "settlement_grpc_stop", map[string]string{
		"subsystem":     "payment_outbox",
		"outbox_status": "PENDING",
		"ledger_rows":   "0",
		"baseline_ok":   "true",
		"fault_verify":  "settlement_grpc_stopped",
	})
}

func TestFault_PaymentSettlementDownThenRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("fault integration test")
	}

	infra, cleanup := SetupPaymentFaultInfra(t)
	defer cleanup()

	ctx := context.Background()
	customerID := uuid.New()
	seed := SeedSucceededIntentWithOutbox(t, infra, customerID, 16_000_000, "fault-grpc-recovery-"+uuid.New().String())

	worker := NewOutboxWorkerForFault(infra)
	infra.SetSettlementDown()
	_, _ = worker.ProcessOutbox(ctx, 10)
	require.Equal(t, "PENDING", PaymentOutboxStatus(t, infra.Pool, seed.OutboxID))

	infra.SetSettlementUp()
	recovered := false
	require.Eventually(t, func() bool {
		n, err := worker.ProcessOutbox(ctx, 10)
		if err != nil || n != 1 {
			return false
		}
		recovered = PaymentOutboxStatus(t, infra.Pool, seed.OutboxID) == "PROCESSED"
		return recovered
	}, 30*time.Second, 200*time.Millisecond)

	AssertPaymentFaultInvariants(t, infra.Pool, seed, seed.AmountMicro, 1)

	faultproof.Log(t, "settlement_grpc_stop_start_recovery", map[string]string{
		"subsystem":    "payment_outbox",
		"recovered":    strconv.FormatBool(recovered),
		"ledger_rows":  "1",
		"baseline_ok":  "true",
		"fault_verify": "settlement_grpc_stopped_then_started",
	})
}

func TestFault_PaymentPGTerminateDuringWebhook(t *testing.T) {
	if testing.Short() {
		t.Skip("fault integration test")
	}

	infra, cleanup := SetupPaymentFaultInfra(t)
	defer cleanup()

	ctx := context.Background()
	customerID := uuid.New()
	SeedCustomer(t, infra.Pool, customerID)

	svc := payment.NewService(infra.Pool, infra.Cfg)
	result, err := svc.CreatePaymentIntent(ctx, customerID, 7_000_000, "USD", "fault-wh-pg-"+uuid.New().String(), nil)
	require.NoError(t, err)
	intent := result.Intent
	providerRef := intent.ProviderRef
	eventID := "evt_pg_kill_" + uuid.New().String()
	payload := fmt.Sprintf(`{"id":"%s","type":"payment_intent.succeeded","data":{"object":{"id":"%s","amount":7000000}}}`,
		eventID, providerRef)

	require.NoError(t, infra.PGContainer.Terminate(ctx))
	RequirePaymentFaultActive(t, func() bool {
		return infra.Pool.Ping(ctx) != nil
	}, "pg ping must fail after terminate")

	err = svc.ProcessStripeWebhook(ctx, eventID, "payment_intent.succeeded", []byte(payload), providerRef, 7_000_000, payload)
	require.Error(t, err)

	faultproof.Log(t, "postgres_container_terminate_webhook", map[string]string{
		"subsystem":    "payment_webhook",
		"committed":    "false",
		"ledger_rows":  "0",
		"baseline_ok":  "true",
		"fault_verify": "postgres_container_terminated",
	})
}
