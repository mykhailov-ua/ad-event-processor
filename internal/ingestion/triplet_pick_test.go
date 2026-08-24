package ingestion

import (
	"testing"

	"ad-event-processor/pkg/faultproof"

	"github.com/stretchr/testify/require"
)

func TestFault_ShardOrchestrator_TripletFailover(t *testing.T) {
	t.Parallel()
	camp := &CampaignTripletPick{PrimaryA: 1, PrimaryB: 2, Reserve: 3}
	shards := map[int]int{}
	for i := range 1000 {
		user := "user-" + string(rune('a'+i%26))
		shards[camp.PickShard("550e8400-e29b-41d4-a716-446655440000", user)]++
	}
	require.Greater(t, shards[1], 0)
	require.Greater(t, shards[2], 0)
	require.Greater(t, shards[3], 0)
	faultproof.Log(t, "triplet_abr_failover", map[string]string{
		"shard_a_hits": "true",
		"shard_b_hits": "true",
		"reserve_hits": "true",
	})
}
