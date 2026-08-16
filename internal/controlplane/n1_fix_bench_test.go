package controlplane

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/controlplane/adminapi"
	"github.com/bidshard/ad-event-processor/internal/database"
	"github.com/bidshard/ad-event-processor/internal/domain"
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
		t.Skip("integration")
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
		t.Skip("integration")
	}
	pool, counter, cleanup := database.SetupTestDBWithQueryCounter(t)
	defer cleanup()
	ctx := context.Background()

	const n = 100
	seedFleetInvariantCustomers(t, pool, n)
	readSvc := adminapi.NewCompositeReadService(pool, nil)

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
		t.Skip("integration")
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
		t.Skip("integration")
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
