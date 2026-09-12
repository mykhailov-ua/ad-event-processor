package campaign

import (
	"context"
	"testing"

	"ad-event-processor/internal/reports"
	"ad-event-processor/internal/testutil"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestCampaignGroup_assignListAndReportScope(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: campaign groups")
	}

	ctx := context.Background()
	cfg := testutil.DefaultPostgresConfig()
	cfg.MigrationDirs = []string{testutil.AdsMigrationsDir(), testutil.BillingMigrationsDir()}
	pool, cleanup := testutil.SetupPostgres(t, cfg)
	defer cleanup()

	customerID := uuid.New()
	campA := uuid.New()
	campB := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO customers (id, name, balance, currency) VALUES ($1, 'cust', 0, 'USD')`, customerID)
	require.NoError(t, err)
	for _, campID := range []uuid.UUID{campA, campB} {
		_, err = pool.Exec(ctx, `
			INSERT INTO campaigns (id, name, budget_limit, status, customer_id, pacing_mode, daily_budget, timezone)
			VALUES ($1, $2, 1000000, 'ACTIVE', $3, 'ASAP', 0, 'UTC')`,
			campID, "camp-"+campID.String()[:8], customerID)
		require.NoError(t, err)
	}

	group, err := CreateCampaignGroup(ctx, pool, CreateCampaignGroupRequest{
		CustomerID: customerID.String(),
		Name:       "keitaro-stream-a",
	})
	require.NoError(t, err)
	groupID, err := uuid.Parse(group.ID)
	require.NoError(t, err)

	assignResp, err := AssignCampaignsToGroup(ctx, pool, groupID, []uuid.UUID{campA, campB})
	require.NoError(t, err)
	require.Len(t, assignResp.Results, 2)
	for _, row := range assignResp.Results {
		require.True(t, row.OK)
	}

	gotGroup, err := GetCampaignGroup(ctx, pool, groupID)
	require.NoError(t, err)
	require.Equal(t, int64(2), gotGroup.MemberCount)

	ids, err := reports.ResolveReportCampaignIDs(ctx, pool, customerID, uuid.Nil, false, groupID, true)
	require.NoError(t, err)
	require.ElementsMatch(t, []uuid.UUID{campA, campB}, ids)

	unassignResp, err := UnassignCampaignsFromGroup(ctx, pool, groupID, []uuid.UUID{campB})
	require.NoError(t, err)
	require.True(t, unassignResp.Results[0].OK)

	ids, err = reports.ResolveReportCampaignIDs(ctx, pool, customerID, uuid.Nil, false, groupID, true)
	require.NoError(t, err)
	require.Equal(t, []uuid.UUID{campA}, ids)

	require.NoError(t, DeleteCampaignGroup(ctx, pool, groupID))
	_, err = GetCampaignGroup(ctx, pool, groupID)
	require.ErrorIs(t, err, ErrCampaignGroupNotFound)
}
