package ingestion

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestSettlementLaneIndex_distribution(t *testing.T) {
	lanes := 8
	counts := make([]int, lanes)
	for i := 0; i < 10_000; i++ {
		id := uuid.New()
		counts[settlementLaneIndex(id, lanes)]++
	}
	for _, c := range counts {
		assert.Greater(t, c, 0, "each lane should receive traffic")
	}
}

func TestSettlementLaneIndex_singleLane(t *testing.T) {
	id := uuid.New()
	assert.Equal(t, 0, settlementLaneIndex(id, 1))
	assert.Equal(t, 0, settlementLaneIndex(id, 0))
}
