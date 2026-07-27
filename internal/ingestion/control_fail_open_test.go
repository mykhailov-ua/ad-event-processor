package ingestion

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestControlFailOpenEnabled_defaultOff(t *testing.T) {
	assert.False(t, ControlFailOpenEnabled(""))
	assert.False(t, ControlFailOpenEnabled("0"))
	assert.False(t, ControlFailOpenEnabled("false"))
	assert.True(t, ControlFailOpenEnabled("1"))
	assert.True(t, ControlFailOpenEnabled("true"))
}

func TestEdgeControlPolicy_conservativeDefault(t *testing.T) {
	assert.True(t, EdgeControlEqualizeWeights(true, false))
	assert.True(t, EdgeControlDrainFrozen(true, false))
	assert.False(t, EdgeControlEqualizeWeights(false, false))
	assert.False(t, EdgeControlDrainFrozen(false, false))
}

func TestEdgeControlPolicy_failOpen(t *testing.T) {
	assert.False(t, EdgeControlEqualizeWeights(true, true))
	assert.False(t, EdgeControlDrainFrozen(true, true))
}
