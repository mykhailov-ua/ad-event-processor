package bpf

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bidshard/ad-event-processor/internal/licensing"
	"github.com/stretchr/testify/require"
)

func TestLoadSealed_InvalidMCK(t *testing.T) {
	var mck [32]byte
	mck[0] = 1
	sealed, err := licensing.SealAsset(sealedEdgeAssetLabel, []byte("not-a-real-elf"), mck)
	require.NoError(t, err)

	dir := t.TempDir()
	blob := filepath.Join(dir, "edge_sealed.bin")
	require.NoError(t, os.WriteFile(blob, sealed, 0o600))
	t.Setenv("AD_EVENT_PROCESSOR_EDGE_SEALED_BLOB", blob)
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_MODE", "enterprise")
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_PATH", filepath.Join(dir, "missing-license.jwt"))

	var objs EdgeObjects
	err = loadEdgeObjectsFromSealed(&objs, nil)
	require.Error(t, err)
}
