// Role: ACCESS EXCLUSIVE lock on campaigns must not block /track hot path (in-memory registry).
// Tier: resilience.
// Infra: testcontainers Postgres (ads schema), Redis x4 via harness; PG lock held in open txn.
// Invariants proved: concurrent track posts return 202 while PG read on campaigns blocks; hot path does not sync-query PG per request.
// Verify: make test-resilience
package resilience_test

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"ad-event-processor/internal/testutil"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Holdout: campaigns table lock must block cold PG read but not /track accept (regression = sync PG on hot path).
func TestFault_TrackHotPathPostgresIsolation(t *testing.T) {
	const workers = 24

	h := setupMultiShardTrackHarness(t, multiShardTrackOpts{
		StreamName:       "resilience-pg-isolation",
		CampaignChannel:  "campaigns:pg-isolation",
		SkipPartitionMgr: true,
	})

	tx, err := h.Pool.Begin(h.ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = tx.Rollback(h.ctx) })

	_, err = tx.Exec(h.ctx, "LOCK TABLE campaigns IN ACCESS EXCLUSIVE MODE")
	require.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(workers)
	for i := range workers {
		shard := i % len(h.CampaignIDs)
		go func(shard int) {
			defer wg.Done()
			status, _ := postClickCampaign(t, h.Handler, h.CampaignIDs[shard], uuid.NewString())
			assert.Equal(t, http.StatusAccepted, status)
		}(shard)
	}
	wg.Wait()

	syncCtx, cancel := context.WithTimeout(h.ctx, 300*time.Millisecond)
	defer cancel()
	var probe int
	err = h.Pool.QueryRow(syncCtx, "SELECT 1 FROM campaigns LIMIT 1").Scan(&probe)
	require.Error(t, err, "cold-path PG read must block while campaigns is locked")

	testutil.LogFaultProof(t, "track_postgres_isolation", map[string]string{
		"harness":         "testcontainers_track_gnet",
		"track_path":      "in_memory_registry",
		"pg_lock":         "campaigns_access_exclusive",
		"track_unblocked": "true",
		"pg_read_blocked": "true",
	})
}
