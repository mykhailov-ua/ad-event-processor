package trafficoptimizer

import (
	"testing"

	"ad-event-processor/internal/flow"

	"github.com/stretchr/testify/require"
)

func TestNewFlowBanditApplier_delegates(t *testing.T) {
	t.Parallel()
	applier := NewFlowBanditApplier()
	require.NotNil(t, applier)
	_, _, err := applier.ApplyThompson([]byte("not-json"), nil, nil, nil, nil, flow.BanditApplyConfig{})
	require.Error(t, err)
}
