package licensing

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWriteFileAtomic_replacesExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "license.jwt")
	require.NoError(t, WriteFileAtomic(path, []byte("token-a"), 0o600))
	require.NoError(t, WriteFileAtomic(path, []byte("token-b"), 0o600))
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "token-b", string(data))
}
