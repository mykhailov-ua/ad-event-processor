package crowdwave

import (
	"testing"

	"ad-event-processor/internal/domain"

	"github.com/stretchr/testify/assert"
)

func synthWaveSimhash(base uint64, delta int) uint64 {
	return base ^ (uint64(delta) << 4)
}

func TestCrowdWave_holdoutSyntheticWaveTriggers(t *testing.T) {
	policy := WavePolicy{
		WindowSec:           3600,
		MinUniqueClusters:   10,
		MinSimhashNeighbors: 8,
		SimhashMaxDist:      2,
	}
	now := int64(1_700_000_000)
	entries := make([]WaveEntry, 10)
	for i := range entries {
		var id [16]byte
		id[0] = byte(i + 1)
		entries[i] = WaveEntry{
			ClusterID: id,
			Simhash:   synthWaveSimhash(0xdeadbeefcafebabe, i%3),
			Observed:  now - int64(i*30),
		}
	}
	state := EvaluateWave(entries, now, policy)
	assert.True(t, state.Active)
	assert.GreaterOrEqual(t, state.UniqueClusters, 10)
	assert.GreaterOrEqual(t, state.SimhashNeighbors, 8)
}

func TestCrowdWave_holdoutOrganicSpreadDoesNotTrigger(t *testing.T) {
	policy := DefaultWavePolicy()
	now := int64(1_700_000_000)
	entries := make([]WaveEntry, 10)
	for i := range entries {
		var id [16]byte
		id[0] = byte(i + 1)
		entries[i] = WaveEntry{
			ClusterID: id,
			Simhash:   uint64(i) * 0x1111111111111111,
			Observed:  now - int64(i*120),
		}
	}
	state := EvaluateWave(entries, now, policy)
	assert.False(t, state.Active)
}

func TestBehaviorSimhash_holdoutStableForSameTemplate(t *testing.T) {
	events := []domain.BehaviorTelemetryEvent{
		{T: "pointerdown"}, {T: "scroll"}, {T: "click"}, {T: "mousemove"},
	}
	h1 := BehaviorSimhash(events)
	h2 := BehaviorSimhash(events)
	assert.Equal(t, h1, h2)
	assert.NotZero(t, h1)
}
