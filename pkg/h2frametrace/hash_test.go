package h2frametrace

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestH2FrameTrace_HashStable(t *testing.T) {
	raw := "settings:0,window_update:0,headers:0,priority:0,priority:0"
	h1, norm, err := HashFromWire(raw)
	require.NoError(t, err)
	h2 := HashTokens(norm)
	assert.Equal(t, h1, h2)
	assert.Equal(t, uint32(0xb0e6d89e), h1)
}

func TestH2FrameTrace_EmptyRejected(t *testing.T) {
	_, _, err := HashFromWire("")
	require.Error(t, err)
}
