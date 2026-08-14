package resilience_test

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/internal/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// /track uses the in-memory registry on the hot path; Postgres lock must not block ingestion.
// Negative control: a direct SELECT on campaigns blocks under the same lock.
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
