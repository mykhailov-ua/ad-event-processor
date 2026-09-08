package tcpsynopt

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTCPSynOpt_HashStable(t *testing.T) {
	h1, norm, err := HashFromWire("nop,nop,mss:1460,sackok,ts")
	require.NoError(t, err)
	assert.Equal(t, "nop,nop,mss:1460,sackok,ts", norm)
	h2, _, err := HashFromWire("NOP,NOP,MSS:1460,SACKOK,TS")
	require.NoError(t, err)
	assert.Equal(t, h1, h2)
}

func TestTCPSynOpt_EmptyRejected(t *testing.T) {
	_, _, err := HashFromWire("")
	require.Error(t, err)
}
