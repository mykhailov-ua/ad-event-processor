//go:build linux

package bpf

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/bidshard/ad-event-processor/internal/edge"
	"github.com/bidshard/ad-event-processor/internal/edge/blocklist"
	"github.com/stretchr/testify/require"
)

func TestPinnedBlocklistOpensAfterPin(t *testing.T) {
	var objs EdgeObjects
	if err := LoadEdgeObjectsForTest(&objs, nil); err != nil {
		t.Skipf("BPF unavailable: %v", err)
	}
	defer objs.Close()

	pinDir := filepath.Join(edge.DefaultBPFPinDir, "pin-dir-smoke-"+strconv.Itoa(os.Getpid()))
	require.NoError(t, os.MkdirAll(pinDir, 0o755))
	t.Cleanup(func() { _ = os.RemoveAll(pinDir) })

	t.Setenv("BPF_PIN_DIR", pinDir)
	t.Setenv("BPF_BLOCKLIST_MAP", "")

	blocklistPath := filepath.Join(pinDir, edge.MapBlocklistV4)
	require.NoError(t, objs.BlocklistV4.Pin(blocklistPath))

	paths := edge.ResolvePinnedMapPaths()
	require.Equal(t, blocklistPath, paths.Blocklist)

	m, err := blocklist.LoadPinnedMap("")
	require.NoError(t, err)
	require.NotNil(t, m)
	m.Close()
}
