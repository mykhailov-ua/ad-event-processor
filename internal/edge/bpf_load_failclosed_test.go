package edge

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadEdgeObjectsLenient_missingSealedBlob(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "missing-edge_sealed.bin")
	t.Setenv("AD_EVENT_PROCESSOR_EDGE_SEALED_BLOB", missing)
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_MODE", "enterprise")

	var objs EdgeObjects
	err := LoadEdgeObjectsLenient(&objs, nil)
	require.Error(t, err)
	require.ErrorIs(t, err, os.ErrNotExist)
}
