package ledger

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/database"
	"github.com/bidshard/ad-event-processor/internal/domain"
	db "github.com/bidshard/ad-event-processor/internal/domain/db"
	leddb "github.com/bidshard/ad-event-processor/internal/ledger/db"
	"github.com/bidshard/ad-event-processor/internal/payment/dbtest"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

const n1InvoiceListSize = 10

func setupLedgerDBWithQueryCounter(t testing.TB) (*pgxpool.Pool, *database.QueryCounter, func()) {
	t.Helper()
	ctx := context.Background()
	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("ledger_n1_db"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("secure_password"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(20*time.Second)),
	)
	require.NoError(t, err)

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	counter := &database.QueryCounter{}
	cfg, err := pgxpool.ParseConfig(connStr)
	require.NoError(t, err)
	cfg.ConnConfig.Tracer = counter

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	require.NoError(t, err)

	dbtest.ApplyMigrations(t, pool, dbtest.RepoRootJoin("internal/ingestion/migrations"))
	dbtest.ApplyMigrations(t, pool, dbtest.RepoRootJoin("internal/ledger/migrations"))

	return pool, counter, func() {
		pool.Close()
		_ = pgContainer.Terminate(ctx)
	}
}

func seedInvoicesWithLines(t testing.TB, pool *pgxpool.Pool, customerID uuid.UUID, n int) {
	t.Helper()
	ctx := context.Background()
	q := leddb.New(pool)
	month := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := range n {
		invID := uuid.New()
		inv, err := q.CreateInvoice(ctx, leddb.CreateInvoiceParams{
			ID:             pgtype.UUID{Bytes: invID, Valid: true},
			CustomerID:     pgtype.UUID{Bytes: customerID, Valid: true},
			BillingMonth:   pgtype.Date{Time: month.AddDate(0, i, 0), Valid: true},
			SubtotalMicro:  int64((i + 1) * 1_000_000),
			TaxMicro:       0,
			TotalMicro:     int64((i + 1) * 1_000_000),
			Currency:       "USD",
			TaxScheme:      leddb.BillingTaxSchemeNONE,
			TaxRateBps:     0,
			LedgerSumMicro: int64((i + 1) * 1_000_000),
		})
		require.NoError(t, err)
		_, err = q.CreateInvoiceLine(ctx, leddb.CreateInvoiceLineParams{
			InvoiceID:   inv.ID,
			LedgerType:  "FEE",
			AmountMicro: int64((i + 1) * 1_000_000),
			EntryCount:  int32(i + 1),
		})
		require.NoError(t, err)
	}
}

func TestN1Fix_ListInvoices_QueryCount(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}
	pool, counter, cleanup := setupLedgerDBWithQueryCounter(t)
	defer cleanup()
	ctx := context.Background()

	customerID := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO customers (id, name, balance, currency) VALUES ($1, 'inv-n1', 0, 'USD')`,
		domain.ToUUID(customerID))
	require.NoError(t, err)
	seedInvoicesWithLines(t, pool, customerID, n1InvoiceListSize)

	svc := NewService(pool)
	rows, err := leddb.New(pool).ListCustomerInvoices(ctx, leddb.ListCustomerInvoicesParams{
		CustomerID: pgtype.UUID{Bytes: customerID, Valid: true},
		Limit:      n1InvoiceListSize,
		Offset:     0,
	})
	require.NoError(t, err)

	counter.Reset()
	for _, row := range rows {
		_, err := svc.invoiceFromDB(ctx, row)
		require.NoError(t, err)
	}
	before := counter.Snapshot()

	counter.Reset()
	_, err = svc.invoicesFromDB(ctx, rows)
	require.NoError(t, err)
	after := counter.Snapshot()

	t.Logf("7_list_invoices queries: before=%d after=%d (invoices=%d)", before, after, n1InvoiceListSize)
	require.Equal(t, int64(n1InvoiceListSize), before)
	require.Equal(t, int64(1), after)
}

const n1LedgerMarginPolicies = 12

func legacyEvaluateLedgerMargin(ctx context.Context, w *Worker, policy *Policy) error {
	if policy == nil {
		return nil
	}
	windowStart := time.Now().Add(-ledgerMarginWindow)
	sums, err := db.New(w.pool).SumCampaignMarginWindow(ctx, db.SumCampaignMarginWindowParams{
		CampaignID: pgtype.UUID{Bytes: policy.CampaignID, Valid: true},
		CreatedAt:  pgtype.Timestamp{Time: windowStart, Valid: true},
	})
	if err != nil {
		return err
	}
	if sums.AdvertiserSpendMicro <= 0 || sums.RtbCostMicro <= 0 {
		return nil
	}
	thresholdBps := CostOverRevenueThresholdBps(policy, w.cfg)
	limitMicro := CostOverRevenueLimitMicro(sums.AdvertiserSpendMicro, thresholdBps)
	if sums.RtbCostMicro <= limitMicro {
		return nil
	}
	var exists bool
	err = w.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM margin_guard_activity
			WHERE campaign_id = $1 AND action = 'pause' AND placement_id = ''
			  AND created_at > now() - INTERVAL '1 hour'
		)`, policy.CampaignID).Scan(&exists)
	if err != nil || exists {
		return err
	}
	return nil
}

func seedLedgerMarginPolicies(t testing.TB, pool *pgxpool.Pool, n int) []*Policy {
	t.Helper()
	ctx := context.Background()
	customerID := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO customers (id, name, balance, currency) VALUES ($1, 'mg-n1', 0, 'USD')`,
		pgtype.UUID{Bytes: customerID, Valid: true})
	require.NoError(t, err)

	windowStart := time.Now().Add(-30 * time.Minute)
	policies := make([]*Policy, 0, n)
	for i := range n {
		campID := uuid.New()
		policyID := uuid.New()
		_, err := pool.Exec(ctx, `
			INSERT INTO campaigns (id, name, budget_limit, current_spend, status, customer_id, pacing_mode, timezone, freq_window)
			VALUES ($1, $2, 100000000, 0, 'ACTIVE', $3, 'ASAP', 'UTC', 86400)`,
			pgtype.UUID{Bytes: campID, Valid: true}, fmt.Sprintf("mg-%d", i), pgtype.UUID{Bytes: customerID, Valid: true})
		require.NoError(t, err)
		_, err = pool.Exec(ctx, `
			INSERT INTO margin_guard_policies (id, campaign_id, name, min_clicks, roi_floor_pct, zero_conv_streak, cost_over_revenue_threshold_bps, is_active)
			VALUES ($1, $2, $3, 0, 0, 0, 500, true)`,
			pgtype.UUID{Bytes: policyID, Valid: true}, pgtype.UUID{Bytes: campID, Valid: true}, fmt.Sprintf("policy-%d", i))
		require.NoError(t, err)
		_, err = pool.Exec(ctx, `
			INSERT INTO balance_ledger (customer_id, campaign_id, amount, type, created_at)
			VALUES ($1, $2, $3, 'FEE', $4)`,
			pgtype.UUID{Bytes: customerID, Valid: true}, pgtype.UUID{Bytes: campID, Valid: true}, -2_000_000, windowStart)
		require.NoError(t, err)
		_, err = pool.Exec(ctx, `
			INSERT INTO balance_ledger (customer_id, campaign_id, amount, type, created_at)
			VALUES ($1, $2, $3, 'rtb_cost', $4)`,
			pgtype.UUID{Bytes: customerID, Valid: true}, pgtype.UUID{Bytes: campID, Valid: true}, 2_500_000, windowStart)
		require.NoError(t, err)
		policies = append(policies, &Policy{ID: policyID, CampaignID: campID, IsActive: true})
	}
	return policies
}

func TestN1Fix_LedgerMarginGuard_QueryCount(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}
	pool, counter, cleanup := database.SetupTestDBWithQueryCounter(t)
	defer cleanup()
	ctx := context.Background()

	policies := seedLedgerMarginPolicies(t, pool, n1LedgerMarginPolicies)
	worker := &Worker{pool: pool, cfg: &config.Config{}}

	counter.Reset()
	for _, policy := range policies {
		require.NoError(t, legacyEvaluateLedgerMargin(ctx, worker, policy))
	}
	before := counter.Snapshot()

	counter.Reset()
	require.NoError(t, worker.evaluateLedgerMarginBatch(ctx, policies))
	after := counter.Snapshot()

	t.Logf("9_ledger_margin queries: before=%d after=%d (policies=%d)", before, after, n1LedgerMarginPolicies)
	require.Equal(t, int64(n1LedgerMarginPolicies*2), before)
	require.Equal(t, int64(3+n1LedgerMarginPolicies), after)
}
