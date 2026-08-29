package ingest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeMarginEconomicsSplit(t *testing.T) {
	split, err := ComputeMarginEconomicsSplit(100_000, 70_000)
	require.NoError(t, err)
	assert.Equal(t, int64(100_000), split.RevenueMicro)
	assert.Equal(t, int64(70_000), split.RtbCostMicro)
	assert.Equal(t, int64(30_000), split.OperatorMarginMicro)
	assert.Equal(t, int64(70_000), split.PublisherPayoutMicro)

	over, err := ComputeMarginEconomicsSplit(50_000, 80_000)
	require.NoError(t, err)
	assert.Equal(t, int64(50_000), over.RtbCostMicro)
	assert.Equal(t, int64(0), over.OperatorMarginMicro)
}
