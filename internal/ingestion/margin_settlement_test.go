package ingestion

import (
	"context"
	"testing"

	"espx/internal/billing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarginSettlement_MultiLegPreservesInvariants(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	infra, cleanup := setupAdsFaultInfra(t)
	defer cleanup()

	ctx := context.Background()
	customerID := uuid.New()
	const balanceMicro = int64(10_000_000)
	_, err := infra.Pool.Exec(ctx,
		"INSERT INTO customers (id, name, balance) VALUES ($1, $2, $3)",
		customerID, "Margin Customer", balanceMicro)
	require.NoError(t, err)

	campaignID := uuid.New()
	const budgetLimit = int64(100_000_000)
	_, err = infra.Pool.Exec(ctx,
		"INSERT INTO campaigns (id, name, status, customer_id, budget_limit) VALUES ($1, $2, $3, $4, $5)",
		campaignID, "Margin Campaign", "ACTIVE", customerID, budgetLimit)
	require.NoError(t, err)

	const spendMicro = int64(100_000)
	const rtbCostMicro = int64(70_000)
	repo := NewCampaignRepoWithDB(infra.Pool, infra.Queries)
	outcomes, err := repo.UpdateSpendBatch(ctx, []SpendFlushItem{{
		CampaignID:   campaignID,
		AmountMicro:  spendMicro,
		RtbCostMicro: rtbCostMicro,
		TxID:         "margin-multi-leg-1",
	}})
	require.NoError(t, err)
	require.Len(t, outcomes, 1)
	require.NoError(t, outcomes[0].Err)

	var feeCount, econCount int
	require.NoError(t, infra.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM balance_ledger WHERE campaign_id = $1 AND type = 'FEE'`, ToUUID(campaignID)).Scan(&feeCount))
	require.NoError(t, infra.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM balance_ledger WHERE campaign_id = $1 AND type IN ('rtb_cost', 'operator_margin', 'publisher_payout')`,
		ToUUID(campaignID)).Scan(&econCount))
	assert.Equal(t, 1, feeCount)
	assert.Equal(t, 3, econCount)

	billing.AssertLedgerBalanceInvariant(t, ctx, infra.Pool, customerID)
	AssertBudgetInvariant(t, ctx, infra.Pool, infra.Redis, campaignID)
}
