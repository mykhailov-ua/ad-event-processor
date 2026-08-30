// Role: Kill one Redis data shard; prove 503 shard_unavailable on killed shard and continued accept on others.
// Tier: resilience.
// Infra: testcontainers Postgres (ads schema), Redis x4 via harness; container stop on shard 2.
// Invariants proved: killed shard 503 with shard_unavailable body; unaffected shards accept under load; AssertBudgetInvariant per healthy shard.
// Verify: make test-resilience
package resilience_test

import (
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	ingestion "ad-event-processor/internal/ingest"
	"ad-event-processor/internal/testutil"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Fail-closed: killed shard must 503 shard_unavailable; healthy shards must keep accepting under concurrent load.
func TestFault_DataShardOutage(t *testing.T) {
	const (
		killedShard = 2
		workers     = 24
		perWorker   = 4
	)

	h := setupMultiShardTrackHarness(t, multiShardTrackOpts{
		StreamName:      "resilience-data-shard-outage",
		CampaignChannel: "campaigns:data-shard-outage",
	})

	for shard, campaignID := range h.CampaignIDs {
		status, _ := postClickCampaign(t, h.Handler, campaignID, uuid.NewString())
		require.Equal(t, http.StatusAccepted, status, "baseline shard %d", shard)
	}

	baselineLatency := h.baselineLatency(1)
	budget := testutil.LatencyBudget(baselineLatency)

	testutil.StopRedisShardContainer(t, h.ShardInfra.Containers[killedShard])
	require.Eventually(t, func() bool {
		return h.ShardInfra.Clients[killedShard].Ping(h.ctx).Err() != nil
	}, 15*time.Second, 100*time.Millisecond, "shard %d must be unreachable", killedShard)

	testutil.TripRedisBreaker(t, h.ShardInfra.Clients[killedShard], h.ShardInfra.Breakers[killedShard])

	statusKilled, bodyKilled := postClickCampaignBody(t, h.Handler, h.CampaignIDs[killedShard], uuid.NewString())
	require.Equal(t, http.StatusServiceUnavailable, statusKilled)
	require.Contains(t, bodyKilled, "shard_unavailable")

	var (
		wg       sync.WaitGroup
		accepted atomic.Int64
	)

	wg.Add(workers)
	for w := range workers {
		worker := w
		go func() {
			defer wg.Done()
			for i := range perWorker {
				shard := (worker + i) % len(h.CampaignIDs)
				if shard == killedShard {
					continue
				}
				status, _ := postClickCampaign(t, h.Handler, h.CampaignIDs[shard], uuid.NewString())
				if status == http.StatusAccepted {
					accepted.Add(1)
				}
			}
		}()
	}
	wg.Wait()

	require.Greater(t, accepted.Load(), int64(workers),
		"unaffected shards must keep accepting under concurrent load")

	for shard, campaignID := range h.CampaignIDs {
		if shard == killedShard {
			status, body := postClickCampaignBody(t, h.Handler, campaignID, uuid.NewString())
			require.Equal(t, http.StatusServiceUnavailable, status)
			require.Contains(t, body, "shard_unavailable")
			continue
		}
		status, elapsed := postClickCampaign(t, h.Handler, campaignID, uuid.NewString())
		require.Equal(t, http.StatusAccepted, status, "shard %d must keep accepting", shard)
		assert.LessOrEqual(t, elapsed, budget, "shard %d latency regression", shard)
		ingestion.AssertBudgetInvariant(t, h.ctx, h.Pool, h.ShardInfra.Clients[shard], campaignID)
	}

	testutil.LogFaultProof(t, "data_shard_outage", map[string]string{
		"harness":           "testcontainers_track_gnet",
		"killed_shard":      "2",
		"unaffected_ok":     "true",
		"concurrent_accept": strconv.FormatInt(accepted.Load(), 10),
		"breaker_tripped":   "true",
		"shard_unavailable": "true",
	})
}
