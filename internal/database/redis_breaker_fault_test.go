package database

import (
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/pkg/faultproof"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Scenario G (milestone §5): promotion/outage on one Redis shard must not open breakers
// on healthy shards (blast radius isolation at breaker layer).
func TestFault_SentinelPromotionIsolation(t *testing.T) {
	const shardCount = 4
	breakers := make([]*RedisBreaker, shardCount)
	for i := range breakers {
		breakers[i] = NewAdaptiveRedisBreaker(50, 2, 50*time.Millisecond, 0.20)
	}

	// Trip breaker on shard 1 only (simulated master kill / promotion window).
	for range 60 {
		breakers[1].RecordFailure()
	}
	require.Equal(t, CircuitOpen, breakers[1].State())
	assert.False(t, breakers[1].Allow())

	healthyAffected := 0
	for _, idx := range []int{0, 2, 3} {
		if breakers[idx].State() != CircuitClosed || !breakers[idx].Allow() {
			healthyAffected++
		}
	}
	require.Equal(t, 0, healthyAffected, "shards 0/2/3 must stay closed while shard 1 is open")

	// Shard 1 recovers: half-open → closed after successes.
	time.Sleep(60 * time.Millisecond)
	require.True(t, breakers[1].Allow())
	breakers[1].RecordSuccess()
	require.True(t, breakers[1].Allow())
	breakers[1].RecordSuccess()
	assert.Equal(t, CircuitClosed, breakers[1].State())

	faultproof.Log(t, "sentinel_promotion_isolation", map[string]string{
		"healthy_shards_affected": "0",
		"failed_shard":            "1",
		"baseline_ok":             "true",
	})
}
