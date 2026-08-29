package worker

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestComputeMABWeights_proportionalCTR(t *testing.T) {
	t.Parallel()
	a := uuid.New()
	b := uuid.New()
	stats := map[uuid.UUID]CreativeMABStat{
		a: {Impressions: 2000, Clicks: 20},
		b: {Impressions: 2000, Clicks: 10},
	}
	weights := ComputeMABWeights(stats)
	assert.Greater(t, weights[a], weights[b])
	assert.Greater(t, weights[b], int32(0))
}
