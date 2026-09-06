package importexport

import (
	"context"
	"fmt"
	"testing"

	"ad-event-processor/internal/campaign"
	"ad-event-processor/internal/database"

	"github.com/stretchr/testify/require"
)

func TestBatchUpsertLandersByNameURL_queryBudgetSublinear(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run bash scripts/ci/query_budget_gate.sh (Docker testcontainers)")
	}

	pool, counter, cleanup := database.SetupTestDBWithQueryCounter(t)
	defer cleanup()
	ctx := context.Background()

	sizes := []int{2, 5}
	points := make([]database.QueryBudgetPoint, 0, len(sizes))
	for _, n := range sizes {
		refs := make(map[string]campaign.CampaignExportLander, n)
		for i := 0; i < n; i++ {
			ref := fmt.Sprintf("l%d", i)
			refs[ref] = campaign.CampaignExportLander{
				Ref:  ref,
				Name: fmt.Sprintf("Lander %d", i),
				URL:  fmt.Sprintf("https://lander-%d.example/", i),
			}
		}
		queries := database.MeasureQueryBudget(counter, func() {
			tx, err := pool.Begin(ctx)
			require.NoError(t, err)
			defer func() { _ = tx.Rollback(ctx) }()
			ids, err := batchUpsertLandersByNameURL(ctx, tx, refs)
			require.NoError(t, err)
			require.Len(t, ids, n)
		})
		points = append(points, database.QueryBudgetPoint{N: n, Queries: queries})
	}
	database.AssertQueryBudgetSublinear(t, "batch_upsert_landers", points, 1)
}

func TestBatchUpsertLandersByNameURL_holdoutQueryBudget(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: batch lander upsert query budget needs Postgres testcontainers")
	}

	pool, counter, cleanup := database.SetupTestDBWithQueryCounter(t)
	defer cleanup()
	ctx := context.Background()

	refs := map[string]campaign.CampaignExportLander{
		"l1": {Ref: "l1", Name: "Lander A", URL: "https://a.example/"},
		"l2": {Ref: "l2", Name: "Lander B", URL: "https://b.example/"},
		"l3": {Ref: "l3", Name: "Lander C", URL: "https://c.example/"},
	}

	tx, err := pool.Begin(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	counter.Reset()
	ids, err := batchUpsertLandersByNameURL(ctx, tx, refs)
	require.NoError(t, err)
	require.Len(t, ids, 3)

	queries := counter.Snapshot()
	t.Logf("batchUpsertLandersByNameURL queries=%d refs=%d", queries, len(refs))
	require.LessOrEqual(t, queries, int64(3), "N+1 regression: lander upsert must not use 2 queries per ref")
}
