package controlplane

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/database"
	"github.com/bidshard/ad-event-processor/internal/domain"
	db "github.com/bidshard/ad-event-processor/internal/domain/db"
	"github.com/bidshard/ad-event-processor/internal/ledger"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

const n1BenchCampaigns = 50

const n1FleetInvariantScanLimit = 500

const n1LedgerInvariantToleranceMicro = int64(1)

func legacyAttachCampaignListMarginBreach(ctx context.Context, s *Service, items []CampaignDTO) {
	for i := range items {
		if items[i].Status != "ACTIVE" {
			continue
		}
		campID, err := uuid.Parse(items[i].ID)
		if err != nil {
			continue
		}
		margin, err := s.GetCampaignMargin(ctx, campID)
		if err != nil {
			continue
		}
		items[i].MarginBreach = margin.MarginBreach
	}
}

func legacyFleetInvariantScan(ctx context.Context, pool *pgxpool.Pool) (bool, error) {
	rows, err := pool.Query(ctx, `SELECT id FROM customers ORDER BY id LIMIT $1`, n1FleetInvariantScanLimit)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var id pgtype.UUID
		if err := rows.Scan(&id); err != nil {
			return false, err
		}
		cid := uuid.UUID(id.Bytes)
		snap, err := ledger.ReadLedgerInvariant(ctx, pool, cid)
		if err != nil {
			return false, err
		}
		diff := snap.BalanceMicro - snap.LedgerSumMicro
		if diff < -n1LedgerInvariantToleranceMicro || diff > n1LedgerInvariantToleranceMicro {
			return false, nil
		}
	}
	return true, rows.Err()
}

func seedMarginBreachCampaigns(t testing.TB, pool *pgxpool.Pool, n int) ([]CampaignDTO, uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	customerID := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO customers (id, name, balance, currency) VALUES ($1, 'n1-bench', 0, 'USD')`,
		domain.ToUUID(customerID))
	require.NoError(t, err)

	windowStart := time.Now().Add(-30 * time.Minute)
	items := make([]CampaignDTO, 0, n)
	for i := range n {
		campID := uuid.New()
		_, err := pool.Exec(ctx, `
			INSERT INTO campaigns (id, name, budget_limit, current_spend, status, customer_id, pacing_mode, timezone, freq_window)
			VALUES ($1, $2, 100000000, 0, 'ACTIVE', $3, 'ASAP', 'UTC', 86400)`,
			domain.ToUUID(campID), fmt.Sprintf("bench-%d", i), domain.ToUUID(customerID))
		require.NoError(t, err)
		_, err = pool.Exec(ctx, `
			INSERT INTO balance_ledger (customer_id, campaign_id, amount, type, created_at)
			VALUES ($1, $2, $3, 'FEE', $4)`,
			domain.ToUUID(customerID), domain.ToUUID(campID), -1_000_000, windowStart)
		require.NoError(t, err)
		_, err = pool.Exec(ctx, `
			INSERT INTO balance_ledger (customer_id, campaign_id, amount, type, created_at)
			VALUES ($1, $2, $3, 'rtb_cost', $4)`,
			domain.ToUUID(customerID), domain.ToUUID(campID), 900_000, windowStart)
		require.NoError(t, err)
		items = append(items, CampaignDTO{
			ID:     campID.String(),
			Status: "ACTIVE",
		})
	}
	return items, customerID
}

func seedFleetInvariantCustomers(t testing.TB, pool *pgxpool.Pool, n int) {
	t.Helper()
	ctx := context.Background()
	for i := range n {
		cid := uuid.New()
		_, err := pool.Exec(ctx, `
			INSERT INTO customers (id, name, balance, currency) VALUES ($1, $2, 1000000, 'USD')`,
			domain.ToUUID(cid), fmt.Sprintf("fleet-%d", i))
		require.NoError(t, err)
		_, err = pool.Exec(ctx, `
			INSERT INTO balance_ledger (customer_id, amount, type, idempotency_hash)
			VALUES ($1, 1000000, 'TOPUP', $2)`,
			domain.ToUUID(cid), fmt.Sprintf("topup-%d", i))
		require.NoError(t, err)
	}
}

func TestN1Fix_MarginBreach_QueryCount(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}
	pool, counter, cleanup := database.SetupTestDBWithQueryCounter(t)
	defer cleanup()
	ctx := context.Background()

	items, _ := seedMarginBreachCampaigns(t, pool, n1BenchCampaigns)
	svc := newBareService(t, pool, nil, &config.Config{})

	counter.Reset()
	legacyItems := append([]CampaignDTO(nil), items...)
	legacyAttachCampaignListMarginBreach(ctx, svc, legacyItems)
	before := counter.Snapshot()

	counter.Reset()
	batchItems := append([]CampaignDTO(nil), items...)
	svc.AttachCampaignListMarginBreach(ctx, batchItems)
	after := counter.Snapshot()

	t.Logf("1_margin_breach queries: before=%d after=%d (campaigns=%d)", before, after, n1BenchCampaigns)
	require.Equal(t, int64(n1BenchCampaigns*2), before)
	require.Equal(t, int64(2), after)
}

func TestN1Fix_FleetInvariant_QueryCount(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}
	pool, counter, cleanup := database.SetupTestDBWithQueryCounter(t)
	defer cleanup()
	ctx := context.Background()

	const n = 100
	seedFleetInvariantCustomers(t, pool, n)
	readSvc := NewCompositeReadService(pool, nil)

	counter.Reset()
	ok, err := legacyFleetInvariantScan(ctx, pool)
	require.NoError(t, err)
	require.True(t, ok)
	before := counter.Snapshot()

	counter.Reset()
	dto, err := readSvc.GetInvariant(ctx, nil)
	require.NoError(t, err)
	require.True(t, dto.OK)
	after := counter.Snapshot()

	t.Logf("3_fleet_invariant queries: before=%d after=%d (customers=%d)", before, after, n)
	require.Equal(t, int64(1+n*2), before)
	require.Equal(t, int64(2), after)
}

func n1BareService(pool *pgxpool.Pool) *Service {
	svc := &Service{cfg: &config.Config{}}
	svc.SetPool(pool)
	return svc
}

func BenchmarkN1Fix_MarginBreach_Legacy(b *testing.B) {
	if testing.Short() {
		b.Skip("integration")
	}
	pool, _, cleanup := database.SetupTestDBWithQueryCounter(b)
	defer cleanup()
	ctx := context.Background()
	items, _ := seedMarginBreachCampaigns(b, pool, n1BenchCampaigns)
	svc := n1BareService(pool)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		batch := append([]CampaignDTO(nil), items...)
		legacyAttachCampaignListMarginBreach(ctx, svc, batch)
	}
}

func BenchmarkN1Fix_MarginBreach_Batched(b *testing.B) {
	if testing.Short() {
		b.Skip("integration")
	}
	pool, _, cleanup := database.SetupTestDBWithQueryCounter(b)
	defer cleanup()
	ctx := context.Background()
	items, _ := seedMarginBreachCampaigns(b, pool, n1BenchCampaigns)
	svc := n1BareService(pool)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		batch := append([]CampaignDTO(nil), items...)
		svc.AttachCampaignListMarginBreach(ctx, batch)
	}
}

func TestN1Fix_BuyerPortfolio_Margin_QueryCount(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}
	pool, counter, cleanup := database.SetupTestDBWithQueryCounter(t)
	defer cleanup()
	ctx := context.Background()

	_, customerID := seedMarginBreachCampaigns(t, pool, n1BenchCampaigns)
	svc := newBareService(t, pool, nil, &config.Config{})

	counter.Reset()
	_, err := svc.GetBuyerPortfolio(ctx, customerID)
	require.NoError(t, err)
	after := counter.Snapshot()

	t.Logf("2_buyer_portfolio queries: after=%d (campaigns=%d)", after, n1BenchCampaigns)
	require.LessOrEqual(t, after, int64(6))
}

func TestN1Fix_Recon_DiscrepancyCustomerLookup(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}
	pool, counter, cleanup := database.SetupTestDBWithQueryCounter(t)
	defer cleanup()
	ctx := context.Background()

	rdb, redisCleanup := database.SetupTestRedis(t)
	defer redisCleanup()

	customerID := uuid.New()
	campaignID := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO customers (id, name, balance, currency) VALUES ($1, 'recon-n1', 0, 'USD')`,
		domain.ToUUID(customerID))
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		INSERT INTO campaigns (id, name, budget_limit, current_spend, status, customer_id, pacing_mode, timezone, freq_window)
		VALUES ($1, 'recon-n1', 100000000, 0, 'ACTIVE', $2, 'ASAP', 'UTC', 86400)`,
		domain.ToUUID(campaignID), domain.ToUUID(customerID))
	require.NoError(t, err)

	start := time.Now().UTC().Truncate(time.Hour).Add(-3 * time.Hour)
	end := start.Add(time.Hour)
	_, err = pool.Exec(ctx, `
		INSERT INTO balance_ledger (customer_id, campaign_id, amount, type, created_at)
		VALUES ($1, $2, $3, 'FEE', $4)`,
		domain.ToUUID(customerID), domain.ToUUID(campaignID), -500_000, start.Add(10*time.Minute))
	require.NoError(t, err)

	syncKey := domain.CampaignSyncKey(campaignID)
	require.NoError(t, rdb.Set(ctx, syncKey, 10_000_000, 0).Err())

	svc := newBareService(t, pool, []redis.UniversalClient{rdb}, &config.Config{QuotaChunkSize: 1_000_000})
	recon := NewReconService(svc)

	counter.Reset()
	require.NoError(t, recon.ReconcileWindow(ctx, start, end))
	after := counter.Snapshot()

	t.Logf("4_recon queries: after=%d (no per-discrepancy customer lookup)", after)
	require.Less(t, after, int64(10))
}

const n1CreditScoringCustomers = 40

func legacyCreditScoringReconLag(ctx context.Context, queries *db.Queries, rows []db.ListCustomersForScoringRow) {
	for _, r := range rows {
		_, _ = queries.MaxCustomerReconLagMicro(ctx, r.ID)
	}
}

func seedCreditScoringCustomers(t testing.TB, pool *pgxpool.Pool, n int) {
	t.Helper()
	ctx := context.Background()
	var runIDVal int64
	err := pool.QueryRow(ctx, `
		INSERT INTO recon_runs (period_start, period_end, status)
		VALUES (NOW() - INTERVAL '1 hour', NOW(), 'COMPLETED')
		RETURNING id`).Scan(&runIDVal)
	require.NoError(t, err)

	for i := range n {
		cid := uuid.New()
		campID := uuid.New()
		_, err := pool.Exec(ctx, `
			INSERT INTO customers (id, name, balance, currency) VALUES ($1, $2, 0, 'USD')`,
			domain.ToUUID(cid), fmt.Sprintf("credit-%d", i))
		require.NoError(t, err)
		_, err = pool.Exec(ctx, `
			INSERT INTO campaigns (id, name, budget_limit, current_spend, status, customer_id, pacing_mode, timezone, freq_window)
			VALUES ($1, $2, 100000000, 0, 'ACTIVE', $3, 'ASAP', 'UTC', 86400)`,
			domain.ToUUID(campID), fmt.Sprintf("credit-camp-%d", i), domain.ToUUID(cid))
		require.NoError(t, err)
		_, err = pool.Exec(ctx, `
			INSERT INTO recon_discrepancies (run_id, campaign_id, customer_id, expected_spend, actual_spend, delta)
			VALUES ($1, $2, $3, 1000, 900, $4)`,
			runIDVal, domain.ToUUID(campID), domain.ToUUID(cid), int64((i%5)+1)*1000)
		require.NoError(t, err)
	}
}

func TestN1Fix_CreditScoringReconLag_QueryCount(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}
	pool, counter, cleanup := database.SetupTestDBWithQueryCounter(t)
	defer cleanup()
	ctx := context.Background()

	seedCreditScoringCustomers(t, pool, n1CreditScoringCustomers)
	queries := db.New(pool)
	rows, err := queries.ListCustomersForScoring(ctx)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(rows), n1CreditScoringCustomers)

	counter.Reset()
	legacyCreditScoringReconLag(ctx, queries, rows)
	before := counter.Snapshot()

	counter.Reset()
	lagRows, err := queries.ListCustomerReconLagMicro(ctx)
	require.NoError(t, err)
	_ = lagRows
	after := counter.Snapshot()

	t.Logf("6_credit_scoring queries: before=%d after=%d (customers=%d)", before, after, len(rows))
	require.Equal(t, int64(len(rows)), before)
	require.Equal(t, int64(1), after)
}

const n1SmartAlertRules = 15

func legacyResolveCampaignIDs(ctx context.Context, pool *pgxpool.Pool, rule smartAlertRuleRow) ([]uuid.UUID, error) {
	if rule.HasCampaign {
		return []uuid.UUID{rule.CampaignID}, nil
	}
	rows, err := pool.Query(ctx, `
		SELECT id FROM campaigns
		WHERE customer_id = $1 AND deleted_at IS NULL`,
		domain.ToUUID(rule.CustomerID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func seedSmartAlertRules(t testing.TB, pool *pgxpool.Pool, customerID uuid.UUID, n int) []smartAlertRuleRow {
	t.Helper()
	ctx := context.Background()
	rules := make([]smartAlertRuleRow, 0, n)
	for i := range n {
		ruleID := uuid.New()
		_, err := pool.Exec(ctx, `
			INSERT INTO alert_rules (
				id, customer_id, name, metric, operator, threshold,
				window_minutes, webhook_url, enabled
			) VALUES ($1, $2, $3, 'clicks', 'gt', 1, 60, 'https://example.com/hook', true)`,
			domain.ToUUID(ruleID), domain.ToUUID(customerID), fmt.Sprintf("rule-%d", i))
		require.NoError(t, err)
		rules = append(rules, smartAlertRuleRow{
			ID:            ruleID,
			CustomerID:    customerID,
			Name:          fmt.Sprintf("rule-%d", i),
			Metric:        "clicks",
			Operator:      "gt",
			Threshold:     1,
			WindowMinutes: 60,
			WebhookURL:    "https://example.com/hook",
		})
	}
	return rules
}

func TestN1Fix_SmartAlertsCampaignResolve_QueryCount(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}
	pool, counter, cleanup := database.SetupTestDBWithQueryCounter(t)
	defer cleanup()
	ctx := context.Background()

	customerID := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO customers (id, name, balance, currency) VALUES ($1, 'alerts-n1', 0, 'USD')`,
		domain.ToUUID(customerID))
	require.NoError(t, err)
	for i := range 3 {
		campID := uuid.New()
		_, err := pool.Exec(ctx, `
			INSERT INTO campaigns (id, name, budget_limit, current_spend, status, customer_id, pacing_mode, timezone, freq_window)
			VALUES ($1, $2, 100000000, 0, 'ACTIVE', $3, 'ASAP', 'UTC', 86400)`,
			domain.ToUUID(campID), fmt.Sprintf("camp-%d", i), domain.ToUUID(customerID))
		require.NoError(t, err)
	}
	rules := seedSmartAlertRules(t, pool, customerID, n1SmartAlertRules)
	worker := &SmartAlertsWorker{svc: &Service{pool: pool}}

	counter.Reset()
	for _, rule := range rules {
		_, err := legacyResolveCampaignIDs(ctx, pool, rule)
		require.NoError(t, err)
	}
	before := counter.Snapshot()

	counter.Reset()
	_, err = worker.batchCampaignIDsByCustomer(ctx, rules)
	require.NoError(t, err)
	after := counter.Snapshot()

	t.Logf("11_smart_alerts queries: before=%d after=%d (rules=%d)", before, after, n1SmartAlertRules)
	require.Equal(t, int64(n1SmartAlertRules), before)
	require.Equal(t, int64(1), after)
}

const n1ReconLedgerIntents = 10

func TestN1Fix_FinancialReconLedger_QueryCount(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}
	pool, counter, cleanup := database.SetupTestDBWithQueryCounter(t)
	defer cleanup()
	ctx := context.Background()

	customerID := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO customers (id, name, balance, currency) VALUES ($1, 'recon-n1', 0, 'USD')`,
		domain.ToUUID(customerID))
	require.NoError(t, err)

	intentIDs := make([]uuid.UUID, 0, n1ReconLedgerIntents)
	for i := range n1ReconLedgerIntents {
		intentID := uuid.New()
		intentIDs = append(intentIDs, intentID)
		_, err := pool.Exec(ctx, `
			INSERT INTO balance_ledger (customer_id, amount, type, payment_intent_id, idempotency_hash)
			VALUES ($1, $2, 'PAYMENT_TOPUP', $3, $4)`,
			domain.ToUUID(customerID), int64((i+1)*1_000_000), domain.ToUUID(intentID), fmt.Sprintf("topup-%d", i))
		require.NoError(t, err)
	}

	cfg := &config.Config{SettlementInternalToken: "recon-n1-token"}
	svc := newBareService(t, pool, nil, cfg)

	counter.Reset()
	for _, intentID := range intentIDs {
		_, _, _, _, _, err := svc.GetLedgerEntry(ctx, intentID)
		require.NoError(t, err)
	}
	before := counter.Snapshot()

	counter.Reset()
	_, err = svc.GetLedgerEntries(ctx, intentIDs)
	require.NoError(t, err)
	after := counter.Snapshot()

	t.Logf("10_financial_recon_ledger queries: before=%d after=%d (intents=%d)", before, after, n1ReconLedgerIntents)
	require.Equal(t, int64(n1ReconLedgerIntents*4), before)
	require.Equal(t, int64(1), after)
}

const n1PostbackStatusConfigs = 20

func legacyListPostbackCampaignStatus(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	rows, err := pool.Query(ctx, `
SELECT
    c.campaign_id::text,
    c.provider,
    (
        SELECT MAX(d.created_at)
        FROM postback_dispatches d
        WHERE d.campaign_id = c.campaign_id AND d.status = 'SENT'
    ) AS last_success_at,
    (
        SELECT COUNT(*)::bigint
        FROM postback_dlq q
        WHERE q.campaign_id = c.campaign_id AND q.status = 'FAILED'
    ) AS dlq_pending_count
FROM postback_configs c
ORDER BY c.campaign_id`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		var campaignID, provider string
		var lastSuccess *time.Time
		var dlqCount int64
		if err := rows.Scan(&campaignID, &provider, &lastSuccess, &dlqCount); err != nil {
			return 0, err
		}
		n++
	}
	return n, rows.Err()
}

func seedPostbackCampaignStatus(t testing.TB, pool *pgxpool.Pool, n int) {
	t.Helper()
	ctx := context.Background()
	for i := range n {
		campID := uuid.New()
		_, err := pool.Exec(ctx, `
			INSERT INTO postback_configs (campaign_id, provider, url_template, api_token_encrypted, target_event)
			VALUES ($1, 'meta', 'https://example.com', '\x00', 'conversion')`,
			domain.ToUUID(campID))
		require.NoError(t, err)
		if i%3 == 0 {
			_, err = pool.Exec(ctx, `
				INSERT INTO postback_dispatches (idempotency_hash, campaign_id, click_id, event_type, status, created_at)
				VALUES ($1, $2, $3, 'conversion', 'SENT', NOW())`,
				fmt.Sprintf("hash-%d", i), domain.ToUUID(campID), fmt.Sprintf("click-%d", i))
			require.NoError(t, err)
		}
		if i%4 == 0 {
			_, err = pool.Exec(ctx, `
				INSERT INTO postback_dlq (outbox_event_id, campaign_id, click_id, event_type, payload, status)
				VALUES ($1, $2, $3, 'conversion', '{}', 'FAILED')`,
				int64(i+1), domain.ToUUID(campID), fmt.Sprintf("click-dlq-%d", i))
			require.NoError(t, err)
		}
	}
}

func TestN1Fix_PostbackCampaignStatus_QueryCount(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}
	pool, counter, cleanup := database.SetupTestDBWithQueryCounter(t)
	defer cleanup()
	ctx := context.Background()

	seedPostbackCampaignStatus(t, pool, n1PostbackStatusConfigs)
	queries := db.New(pool)

	counter.Reset()
	legacyN, err := legacyListPostbackCampaignStatus(ctx, pool)
	require.NoError(t, err)
	before := counter.Snapshot()

	counter.Reset()
	rows, err := queries.ListPostbackCampaignStatus(ctx)
	require.NoError(t, err)
	after := counter.Snapshot()

	t.Logf("12_postback_status queries: before=%d after=%d (configs=%d rows=%d)", before, after, n1PostbackStatusConfigs, len(rows))
	require.Equal(t, n1PostbackStatusConfigs, legacyN)
	require.Len(t, rows, n1PostbackStatusConfigs)
	require.Equal(t, int64(1), before)
	require.Equal(t, int64(1), after)
}

const n1TeamMembers = 15

func legacyListTeamCampaignCounts(ctx context.Context, pool *pgxpool.Pool, customerID uuid.UUID) (int, error) {
	rows, err := pool.Query(ctx, `
		SELECT u.id,
			(SELECT COUNT(*)::bigint FROM campaigns c
			 WHERE c.customer_id = u.customer_id AND c.owner_user_id = u.id) AS campaigns_owned
		FROM users u
		WHERE u.customer_id = $1
		ORDER BY u.email`, domain.ToUUID(customerID))
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		var userID uuid.UUID
		var owned int64
		if err := rows.Scan(&userID, &owned); err != nil {
			return 0, err
		}
		n++
	}
	return n, rows.Err()
}

func listTeamCampaignCounts(ctx context.Context, pool *pgxpool.Pool, customerID uuid.UUID) (int, error) {
	rows, err := pool.Query(ctx, `
		SELECT u.id, COALESCE(cc.campaigns_owned, 0)
		FROM users u
		LEFT JOIN (
			SELECT owner_user_id, COUNT(*)::bigint AS campaigns_owned
			FROM campaigns
			WHERE customer_id = $1
			GROUP BY owner_user_id
		) cc ON cc.owner_user_id = u.id
		WHERE u.customer_id = $1
		ORDER BY u.email`, domain.ToUUID(customerID))
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		var userID uuid.UUID
		var owned int64
		if err := rows.Scan(&userID, &owned); err != nil {
			return 0, err
		}
		n++
	}
	return n, rows.Err()
}

func seedTeamMembersWithCampaigns(t testing.TB, pool *pgxpool.Pool, n int) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY,
			email TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			role TEXT NOT NULL,
			customer_id UUID,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			is_blocked BOOLEAN NOT NULL DEFAULT FALSE,
			email_verified BOOLEAN NOT NULL DEFAULT TRUE
		)`)
	require.NoError(t, err)
	customerID := uuid.New()
	_, err = pool.Exec(ctx, `
		INSERT INTO customers (id, name, balance, currency) VALUES ($1, 'team-n1', 0, 'USD')`,
		domain.ToUUID(customerID))
	require.NoError(t, err)
	for i := range n {
		userID := uuid.New()
		_, err := pool.Exec(ctx, `
			INSERT INTO users (id, email, password_hash, role, customer_id)
			VALUES ($1, $2, 'hash', 'MEDIA_BUYER', $3)`,
			userID, fmt.Sprintf("buyer-%d@example.com", i), customerID)
		require.NoError(t, err)
		for j := range i % 4 {
			campID := uuid.New()
			_, err := pool.Exec(ctx, `
				INSERT INTO campaigns (id, name, status, customer_id, budget_limit, current_spend, owner_user_id)
				VALUES ($1, $2, 'ACTIVE', $3, 1000000, 0, $4)`,
				domain.ToUUID(campID), fmt.Sprintf("camp-%d-%d", i, j), domain.ToUUID(customerID), userID)
			require.NoError(t, err)
		}
	}
	return customerID
}

func TestN1Fix_TeamOverviewCampaignCount_QueryCount(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}
	pool, counter, cleanup := database.SetupTestDBWithQueryCounter(t)
	defer cleanup()
	ctx := context.Background()

	customerID := seedTeamMembersWithCampaigns(t, pool, n1TeamMembers)

	counter.Reset()
	legacyN, err := legacyListTeamCampaignCounts(ctx, pool, customerID)
	require.NoError(t, err)
	before := counter.Snapshot()

	counter.Reset()
	joinedN, err := listTeamCampaignCounts(ctx, pool, customerID)
	require.NoError(t, err)
	after := counter.Snapshot()

	t.Logf("13_team_campaign_count queries: before=%d after=%d (members=%d)", before, after, n1TeamMembers)
	require.Equal(t, n1TeamMembers, legacyN)
	require.Equal(t, n1TeamMembers, joinedN)
	require.Equal(t, int64(1), before)
	require.Equal(t, int64(1), after)
}
