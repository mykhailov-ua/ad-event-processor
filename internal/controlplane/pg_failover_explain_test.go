package controlplane

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"ad-event-processor/internal/database"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func TestExplainAudit_PgFailoverQueries(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: pg failover EXPLAIN (run make test-integration)")
	}

	ctx := context.Background()
	pool, cleanup := database.SetupTestDB(t)
	defer cleanup()

	ledgerRows := explainFailoverLedgerScale()
	seedPgFailoverExplainDataWithScale(t, ctx, pool, ledgerRows)

	custID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	campID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	knownHash := "explain-hash-25000"
	newHash := fmt.Sprintf("pg-failover-explain-%d", time.Now().UnixNano())
	auditSince := time.Now().Add(-time.Hour)
	pageSize := 5000
	var pageAfterID int64

	type queryCase struct {
		name    string
		sql     string
		args    []any
		hotPath bool
	}

	queries := []queryCase{
		{
			name:    "failover.GetLedgerByHashForUpdate",
			hotPath: true,
			sql: `EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT)
SELECT * FROM balance_ledger WHERE idempotency_hash = $1 FOR UPDATE`,
			args: []any{knownHash},
		},
		{
			name: "failover.CreateLedgerEntry",
			sql: `EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT)
INSERT INTO balance_ledger (customer_id, campaign_id, amount, type, idempotency_hash)
VALUES ($1, $2, $3, 'TOPUP', $4) RETURNING id`,
			args: []any{custID, campID, int64(1_000_000), newHash},
		},
		{
			name: "failover.countLedgerDuplicatesTimeBounded",
			sql: `EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT)
SELECT COUNT(*) FROM (
	SELECT idempotency_hash FROM balance_ledger
	WHERE idempotency_hash IS NOT NULL AND created_at >= $1
	GROUP BY idempotency_hash HAVING COUNT(*) > 1
) d`,
			args: []any{auditSince},
		},
		{
			name: "failover.syncBalanceLedgerPaginated",
			sql: `EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT)
SELECT id, customer_id, campaign_id, amount, type::text, idempotency_hash, payment_intent_id, created_at
FROM balance_ledger
WHERE id > $1
ORDER BY id ASC
LIMIT $2`,
			args: []any{pageAfterID, pageSize},
		},
		{
			name: "failover.syncCustomersSnapshot",
			sql: `EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT)
SELECT id, name, balance, currency FROM customers`,
		},
		{
			name:    "failover.UpdateCustomerBalanceManagement",
			hotPath: true,
			sql: `EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT)
UPDATE customers SET balance = balance + $2, updated_at = NOW() WHERE id = $1`,
			args: []any{custID, int64(1_000_000)},
		},
	}

	t.Logf("=== PG failover EXPLAIN audit (ledger_rows=%d, page_size=%d) ===", ledgerRows, pageSize)

	var summaries []string
	warnCount := 0
	for _, qc := range queries {
		rows, err := pool.Query(ctx, qc.sql, qc.args...)
		require.NoError(t, err, qc.name)
		raw, err := collectExplainText(rows)
		rows.Close()
		require.NoError(t, err, qc.name)

		plan := database.ParseExplainPlan(raw)
		findings := database.AnalyzeExplainPlan(qc.name, plan, qc.hotPath, 500)
		for _, f := range findings {
			if f.Severity == "warn" {
				warnCount++
				t.Logf("WARN [%s] %s", f.Query, f.Message)
			}
		}
		summaries = append(summaries, fmt.Sprintf("%s: exec=%.3fms plan=%.3fms nodes=%d findings=%d",
			qc.name, plan.ExecutionTimeMS, plan.PlanningTimeMS, len(plan.Nodes), len(findings)))
		t.Logf("--- %s ---\n%s", qc.name, raw)
	}

	t.Log("--- PG FAILOVER EXPLAIN SUMMARY ---")
	for _, s := range summaries {
		t.Log(s)
	}
	t.Logf("ledger_rows=%d warnings=%d", ledgerRows, warnCount)

	require.Equal(t, 0, warnCount, "failover hot-path queries must have zero warn-level plan findings")
}

func explainFailoverLedgerScale() int {
	if raw := os.Getenv("PG_FAILOVER_EXPLAIN_LEDGER_ROWS"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			return n
		}
	}
	return 50_000
}

func seedPgFailoverExplainDataWithScale(t *testing.T, ctx context.Context, pool *pgxpool.Pool, ledgerRows int) {
	t.Helper()
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			if !strings.Contains(err.Error(), "already exists") && !strings.Contains(err.Error(), "duplicate") {
				t.Fatalf("seed: %v\nsql: %s", err, sql)
			}
		}
	}

	customers := ledgerRows / 100
	if customers < 100 {
		customers = 100
	}
	campaigns := ledgerRows / 10
	if campaigns < 1000 {
		campaigns = 1000
	}

	exec(`INSERT INTO customers (id, name, balance, currency)
SELECT ('00000000-0000-4000-8000-' || lpad(to_hex(g), 12, '0'))::uuid, 'cust-' || g, (g % 100) * 1000000, 'USD'
FROM generate_series(1, $1) g ON CONFLICT DO NOTHING`, customers)

	exec(`INSERT INTO campaigns (id, name, status, budget_limit, current_spend, customer_id, pacing_mode, timezone)
SELECT ('00000000-0000-4000-8000-' || lpad(to_hex(g), 12, '0'))::uuid,
  'camp-' || g, 'ACTIVE', 100000000, 0,
  ('00000000-0000-4000-8000-' || lpad(to_hex(1 + (g % $1)), 12, '0'))::uuid,
  'ASAP'::pacing_mode_type, 'UTC'
FROM generate_series(1, $2) g ON CONFLICT DO NOTHING`, customers, campaigns)

	exec(`INSERT INTO balance_ledger (customer_id, campaign_id, amount, type, idempotency_hash)
SELECT ('00000000-0000-4000-8000-' || lpad(to_hex(1 + (g % $1)), 12, '0'))::uuid,
  ('00000000-0000-4000-8000-' || lpad(to_hex(1 + (g % $2)), 12, '0'))::uuid,
  (g % 10) * 100000,
  (ARRAY['FEE','TOPUP','PAYMENT_TOPUP','RELEASE']::ledger_type[])[1 + (g % 4)],
  'explain-hash-' || g
FROM generate_series(1, $3) g ON CONFLICT DO NOTHING`, customers, campaigns, ledgerRows)

	exec(`ANALYZE customers`)
	exec(`ANALYZE campaigns`)
	exec(`ANALYZE balance_ledger`)
}

func collectExplainText(rows pgxRows) (string, error) {
	var b strings.Builder
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			return "", err
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String(), rows.Err()
}

type pgxRows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}
