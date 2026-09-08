package track

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWasmAttestLoader_holdout_noFetchOnTrackPixel(t *testing.T) {
	src := string(WasmAttestLoaderJS)
	require.Contains(t, src, "aad_pow_solve")
	require.Contains(t, src, "WebAssembly.instantiate")
	require.NotContains(t, src, "/track HTTP")
}

func TestWasmAttestLoader_holdout_exportsGlobal(t *testing.T) {
	src := string(WasmAttestLoaderJS)
	require.Contains(t, src, "globalThis.aedWasmAttest")
	require.True(t, strings.Index(src, "fetch") < strings.Index(src, "solvePoW"))
}
