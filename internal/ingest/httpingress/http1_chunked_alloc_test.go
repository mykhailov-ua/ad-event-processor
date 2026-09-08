package httpingress

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func chunkedBodyOffset(wire []byte) int {
	for i := 0; i+3 < len(wire); i++ {
		if wire[i] == '\r' && wire[i+1] == '\n' && wire[i+2] == '\r' && wire[i+3] == '\n' {
			return i + 4
		}
	}
	return -1
}

func TestParseHTTP1ChunkedBodyFragmentedAllocs(t *testing.T) {
	wire := FragmentedChunkedOpenRTBRequest()
	off := chunkedBodyOffset(wire)
	require.Greater(t, off, 0)

	var scratch []byte
	allocs := testing.AllocsPerRun(100, func() {
		ResetChunkScratch(&scratch)
		_, _, _, err := ParseHTTP1ChunkedBody(wire, off, 1<<20, 0, &scratch)
		require.NoError(t, err)
	})
	assert.Equal(t, float64(0), allocs)
}
