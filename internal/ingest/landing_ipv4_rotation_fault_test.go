package ingest

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFault_IPv4RotationShadow_BudgetInvariant(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: fault test (run make test-integration)")
	}

	ctx := context.Background()
	infra, cleanup := setupAdsFaultInfra(t)
	defer cleanup()

	stack := startAdsIngestStackOpts(t, infra, "ads-fault-ipv4-rotation", adsIngestStackOpts{
		filterTimeoutMs:   2000,
		maxWorkers:        4,
		rateLimit:         1_000_000,
		productionFilters: true,
	})
	defer stack.Close(t)

	table := NewIPv4RotationTable()
	table.SetMode("shadow")
	table.SetPolicy(uint64(time.Minute.Nanoseconds()), 2)
	stack.Handler.ConfigureIPv4Rotation(table)

	for i := 1; i <= 4; i++ {
		conn := serveClickFromIPUser(stack.Handler, stack.CampaignID, rotatedIPv4Host(i), "sticky-user")
		require.NotContains(t, string(conn.Written()), "X-ad-event-processor-Safe-View: l1v4")
	}

	AssertBudgetInvariant(t, ctx, infra.Pool, infra.Redis, stack.CampaignID)
}
