package payment

import (
	"espx/pkg/faultproof"

	"context"
	"sync"
	"testing"
	"time"

	"espx/internal/ingestion"
	"espx/internal/payment/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func newReconForFault(infra *paymentFaultInfra) *ReconService {
	return NewReconService(infra.Pool, NewSettlementLedgerClient(infra.Cfg), nil)
}

func countReconFindingsByKind(t *testing.T, pool *pgxpool.Pool, runID int64, kind db.PaymentFinancialFindingKind) int {
	t.Helper()
	var n int
	err := pool.QueryRow(context.Background(), `
		SELECT COUNT(*) FROM payment.financial_recon_findings
		WHERE run_id = $1 AND kind = $2`, runID, kind).Scan(&n)
	require.NoError(t, err)
	return n
}

func TestFault_FinancialReconCleanSettlement(t *testing.T) {
	if testing.Short() {
		t.Skip("fault integration test")
	}

	infra, cleanup := setupPaymentFaultInfra(t)
	defer cleanup()

	seedSettledIntent(t, infra, uuid.New(), 18_000_000, "fault-recon-clean-"+uuid.New().String())

	recon := newReconForFault(infra)
	end := time.Now().UTC()
	summary, err := recon.Run(context.Background(), end.Add(-time.Hour), end)
	require.NoError(t, err)
	require.Equal(t, 0, summary.FindingsCount)
	require.GreaterOrEqual(t, summary.IntentsChecked, 1)

	faultproof.Log(t, "financial_recon_clean_settlement", map[string]string{
		"subsystem":       "payment_financial_recon",
		"findings":        "0",
		"intents_checked": itoaPaymentFault(summary.IntentsChecked),
		"baseline_ok":     "true",
		"fault_type":      "none",
	})
}

func TestFault_FinancialReconMissingTopup(t *testing.T) {
	if testing.Short() {
		t.Skip("fault integration test")
	}

	infra, cleanup := setupPaymentFaultInfra(t)
	defer cleanup()

	seedSucceededIntentWithOutbox(t, infra, uuid.New(), 12_000_000, "fault-recon-miss-"+uuid.New().String())

	recon := newReconForFault(infra)
	end := time.Now().UTC()
	summary, err := recon.Run(context.Background(), end.Add(-time.Hour), end)
	require.NoError(t, err)
	require.GreaterOrEqual(t, summary.TopupMissing, 1)
	require.Equal(t, 1, countReconFindingsByKind(t, infra.Pool, summary.RunID, db.PaymentFinancialFindingKindMISSINGLEDGERTOPUP))

	faultproof.Log(t, "financial_recon_missing_topup", map[string]string{
		"subsystem":     "payment_financial_recon",
		"findings":      itoaPaymentFault(summary.FindingsCount),
		"topup_missing": itoaPaymentFault(summary.TopupMissing),
		"baseline_ok":   "true",
		"fault_type":    "missing_topup",
	})
}

func TestFault_FinancialReconDeadOutbox(t *testing.T) {
	if testing.Short() {
		t.Skip("fault integration test")
	}

	infra, cleanup := setupPaymentFaultInfra(t)
	defer cleanup()

	ctx := context.Background()
	customerID := uuid.New()
	seed := seedSucceededIntentWithOutbox(t, infra, customerID, 9_000_000, "fault-recon-dead-"+uuid.New().String())
	_, err := infra.Pool.Exec(ctx, `DELETE FROM payment.payment_outbox WHERE event_type = 'SETTLE_BALANCE'`)
	require.NoError(t, err)

	svc := NewService(infra.Pool, NewMockProvider(), infra.Cfg)
	processRefundWebhook(t, infra.Pool, svc, "evt_recon_dead_"+uuid.New().String(), seed.ProviderRef, "re_recon_dead_"+uuid.New().String(), 9_000_000)

	worker := newOutboxWorkerForFault(infra)
	n, err := worker.ProcessOutbox(ctx, 10)
	require.NoError(t, err)
	require.Equal(t, 0, n)

	recon := newReconForFault(infra)
	end := time.Now().UTC()
	summary, err := recon.Run(ctx, end.Add(-time.Hour), end)
	require.NoError(t, err)
	require.GreaterOrEqual(t, summary.DeadOutboxRows, 1)
	require.Equal(t, 1, countReconFindingsByKind(t, infra.Pool, summary.RunID, db.PaymentFinancialFindingKindDEADOUTBOX))

	faultproof.Log(t, "financial_recon_dead_outbox", map[string]string{
		"subsystem":   "payment_financial_recon",
		"dead_outbox": itoaPaymentFault(summary.DeadOutboxRows),
		"findings":    itoaPaymentFault(summary.FindingsCount),
		"baseline_ok": "true",
		"fault_type":  "dead_outbox",
	})
}

func TestFault_FinancialReconRefundDrift(t *testing.T) {
	if testing.Short() {
		t.Skip("fault integration test")
	}

	infra, cleanup := setupPaymentFaultInfra(t)
	defer cleanup()

	customerID := uuid.New()
	seed := seedSettledIntent(t, infra, customerID, 20_000_000, "fault-recon-drift-"+uuid.New().String())
	svc := NewService(infra.Pool, NewMockProvider(), infra.Cfg)
	processRefundWebhook(t, infra.Pool, svc, "evt_recon_drift_"+uuid.New().String(), seed.ProviderRef, "re_recon_drift_"+uuid.New().String(), 6_000_000)

	recon := newReconForFault(infra)
	end := time.Now().UTC()
	summary, err := recon.Run(context.Background(), end.Add(-time.Hour), end)
	require.NoError(t, err)
	require.Equal(t, 1, countReconFindingsByKind(t, infra.Pool, summary.RunID, db.PaymentFinancialFindingKindREFUNDLEDGERDRIFT))

	faultproof.Log(t, "financial_recon_refund_drift", map[string]string{
		"subsystem":   "payment_financial_recon",
		"findings":    itoaPaymentFault(summary.FindingsCount),
		"drift_kind":  "REFUND_LEDGER_DRIFT",
		"baseline_ok": "true",
		"fault_type":  "ledger_drift",
	})
}

func TestFault_FinancialReconSettlementFailedIntent(t *testing.T) {
	if testing.Short() {
		t.Skip("fault integration test")
	}

	infra, cleanup := setupPaymentFaultInfra(t)
	defer cleanup()

	ctx := context.Background()
	customerID := uuid.New()
	_ = seedSucceededIntentWithOutbox(t, infra, customerID, 9_000_000, "fault-recon-fail-"+uuid.New().String())
	_, err := infra.Pool.Exec(ctx, `DELETE FROM customers WHERE id = $1`, ingestion.ToUUID(customerID))
	require.NoError(t, err)

	worker := newOutboxWorkerForFault(infra)
	n, err := worker.ProcessOutbox(ctx, 10)
	require.NoError(t, err)
	require.Equal(t, 0, n)

	recon := newReconForFault(infra)
	end := time.Now().UTC()
	summary, err := recon.Run(ctx, end.Add(-time.Hour), end)
	require.NoError(t, err)
	require.GreaterOrEqual(t, summary.SettlementFailed, 1)
	require.Equal(t, 1, countReconFindingsByKind(t, infra.Pool, summary.RunID, db.PaymentFinancialFindingKindSETTLEMENTFAILEDINTENT))

	faultproof.Log(t, "financial_recon_settlement_failed", map[string]string{
		"subsystem":         "payment_financial_recon",
		"settlement_failed": itoaPaymentFault(summary.SettlementFailed),
		"findings":          itoaPaymentFault(summary.FindingsCount),
		"baseline_ok":       "true",
		"fault_type":        "settlement_failed",
	})
}

func TestFault_FinancialReconConcurrentRuns(t *testing.T) {
	if testing.Short() {
		t.Skip("fault integration test")
	}

	infra, cleanup := setupPaymentFaultInfra(t)
	defer cleanup()

	seedSettledIntent(t, infra, uuid.New(), 15_000_000, "fault-recon-conc-"+uuid.New().String())
	recon := newReconForFault(infra)
	end := time.Now().UTC()
	start := end.Add(-time.Hour)

	const workers = 4
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			_, _ = recon.Run(context.Background(), start, end)
		}()
	}
	wg.Wait()

	var runCount int
	require.NoError(t, infra.Pool.QueryRow(context.Background(), `
		SELECT COUNT(*) FROM payment.financial_recon_runs WHERE status = 'COMPLETED'`).Scan(&runCount))
	require.Equal(t, workers, runCount)

	faultproof.Log(t, "financial_recon_concurrent_runs", map[string]string{
		"subsystem":   "payment_financial_recon",
		"workers":     "4",
		"runs":        itoaPaymentFault(runCount),
		"baseline_ok": "true",
		"fault_type":  "concurrency_stress",
	})
}

func TestFault_FinancialReconOpsAlert(t *testing.T) {
	if testing.Short() {
		t.Skip("fault integration test")
	}

	infra, cleanup := setupPaymentFaultInfra(t)
	defer cleanup()

	stub := &stubPaymentNotifierClient{}
	cfg := testPaymentOpsConfig()
	alerter := NewFinancialReconAlerter(&NotifierClient{client: stub}, cfg)
	require.NotNil(t, alerter)

	seedSucceededIntentWithOutbox(t, infra, uuid.New(), 11_000_000, "fault-recon-ops-"+uuid.New().String())
	recon := NewReconService(infra.Pool, NewSettlementLedgerClient(infra.Cfg), alerter)

	end := time.Now().UTC()
	summary, err := recon.Run(context.Background(), end.Add(-time.Hour), end)
	require.NoError(t, err)
	require.GreaterOrEqual(t, summary.FindingsCount, 1)

	time.Sleep(200 * time.Millisecond)
	requests := stub.snapshot()
	require.Len(t, requests, 1)
	require.NotEmpty(t, requests[0].DedupKey)
	require.Contains(t, requests[0].Body, "MISSING_LEDGER_TOPUP")

	faultproof.Log(t, "financial_recon_ops_alert", map[string]string{
		"subsystem":   "payment_financial_recon",
		"findings":    itoaPaymentFault(summary.FindingsCount),
		"notified":    "true",
		"baseline_ok": "true",
		"fault_type":  "missing_topup",
	})
}
