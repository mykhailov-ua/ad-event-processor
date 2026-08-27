package edge

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEdgeDecoy_notUsedBySealedLoader(t *testing.T) {
	stub := decoyEdgeBPFStub()
	require.NotEmpty(t, stub)
	require.Equal(t, byte(0x7f), stub[0])
	require.Equal(t, byte('E'), stub[1])

	raw, err := os.ReadFile("bpf_load_sealed.go")
	require.NoError(t, err)
	body := string(raw)
	require.NotContains(t, body, "decoyEdgeBPFStub")
	require.NotContains(t, body, "decoyEdgeBPFEmbed")
}
