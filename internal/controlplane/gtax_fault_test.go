package controlplane

import (
	"github.com/bidshard/ad-event-processor/pkg/faultproof"

	"context"
	"strconv"
	"testing"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/database"
	"github.com/bidshard/ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func ensureBillingCTVProfileSchema(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	_, err := pool.Exec(ctx, `
		CREATE SCHEMA IF NOT EXISTS billing;
		DO $$ BEGIN
			CREATE TYPE billing.tax_scheme AS ENUM ('NONE', 'VAT', 'SALES_TAX');
		EXCEPTION WHEN duplicate_object THEN NULL;
		END $$;
		CREATE TABLE IF NOT EXISTS billing.customer_tax_profiles (
			customer_id UUID PRIMARY KEY,
			country_code CHAR(2) NOT NULL DEFAULT 'US',
			tax_region TEXT,
			tax_scheme billing.tax_scheme NOT NULL DEFAULT 'NONE',
			tax_rate_bps INT NOT NULL DEFAULT 0,
			ctv_gtax_enabled BOOLEAN NOT NULL DEFAULT FALSE,
			ctv_gtax_rate_bps INT NOT NULL DEFAULT 0,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`)
	require.NoError(t, err)
}

func TestFault_CTVGtaxSettlementReplay(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	ctx := context.Background()
	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	ensureBillingCTVProfileSchema(t, pool)
	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	customerID := uuid.New()
	campaignID := uuid.New()
	const budgetLimit = int64(50_000_000)
	const spendMicro = int64(1_000_000)

	_, err := pool.Exec(ctx,
		"INSERT INTO customers (id, name, balance) VALUES ($1, $2, $3)",
		domain.ToUUID(customerID), "CTV Customer", 200_000_000)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		"INSERT INTO campaigns (id, name, status, customer_id, budget_limit) VALUES ($1, $2, $3, $4, $5)",
		domain.ToUUID(campaignID), "CTV Campaign", "ACTIVE", domain.ToUUID(customerID), budgetLimit)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		INSERT INTO billing.customer_tax_profiles (
			customer_id, country_code, tax_scheme, tax_rate_bps, ctv_gtax_enabled, ctv_gtax_rate_bps
		) VALUES ($1, 'US', 'SALES_TAX', 725, TRUE, 500)`, domain.ToUUID(customerID))
	require.NoError(t, err)
	require.NoError(t, rdb.Set(ctx, domain.BudgetCampaignKey(campaignID), budgetLimit, 0).Err())

	cfg := &config.Config{SettlementInternalToken: "gtax-test-token"}
	svc := NewService(context.Background(), pool, []redis.UniversalClient{rdb}, nil, cfg)
	defer svc.Close()
	handler := NewSettlementHandler(svc, cfg)

	settlementID := "ctv-settle-" + uuid.New().String()

	var first domain.CTVSettlementResult
	for i := range 3 {
		resp, callErr := handler.applyCTVSettlement(ctx, settlementID, customerID, campaignID, spendMicro)
		require.NoError(t, callErr)
		if i == 0 {
			first = resp
			require.True(t, resp.Applied)
			require.Equal(t, int64(50_000), resp.TaxMicro)
		} else {
			require.False(t, resp.Applied)
			require.Equal(t, first.FeeLedgerID, resp.FeeLedgerID)
			require.Equal(t, first.TaxLedgerID, resp.TaxLedgerID)
		}
	}

	var settlementRows int
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM ctv_gtax_settlements WHERE settlement_id = $1`, settlementID).Scan(&settlementRows))
	require.Equal(t, 1, settlementRows)

	var outboxRows int
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM outbox_events WHERE event_type = 'APPLY_GTV_SETTLEMENT'`).Scan(&outboxRows))
	require.Equal(t, 1, outboxRows)

	var feeRows, taxRows int
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM balance_ledger WHERE campaign_id = $1 AND type = 'FEE'`, domain.ToUUID(campaignID)).Scan(&feeRows))
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM balance_ledger WHERE campaign_id = $1 AND type = 'CTV_GTAX'`, domain.ToUUID(campaignID)).Scan(&taxRows))
	require.Equal(t, 1, feeRows)
	require.Equal(t, 1, taxRows)

	remaining := budgetLimit - spendMicro
	require.NoError(t, rdb.Set(ctx, domain.BudgetCampaignKey(campaignID), remaining, 0).Err())
	domain.AssertBudgetInvariant(t, ctx, pool, rdb, campaignID)

	faultproof.Log(t, "gtax_settlement_replay", map[string]string{
		"fault":         "gtax_settlement_replay",
		"proposal_rows": "1",
		"settlement_id": settlementID,
		"fee_ledger_id": strconv.FormatInt(first.FeeLedgerID, 10),
		"tax_micro":     strconv.FormatInt(first.TaxMicro, 10),
	})
}
