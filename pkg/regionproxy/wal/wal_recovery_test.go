package wal

import (
	"os"
	"testing"

	"ad-event-processor/pkg/iogate"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWAL_RecoverTruncatesTornTail(t *testing.T) {
	dir := t.TempDir()
	gate := iogate.NewDiskWriteGate(iogate.TestGateConfig())

	w, err := Open(dir, gate)
	require.NoError(t, err)

	for i := range 5 {
		_, err := w.Append([]byte{byte('a' + i)})
		require.NoError(t, err)
	}
	cleanPos := w.WritePos()
	require.NoError(t, w.Close())

	f, err := os.OpenFile(w.path, os.O_RDWR, 0o644)
	require.NoError(t, err)
	_, err = f.WriteAt([]byte{0xFF, 0xFF, 0xFF, 0xFF, 0x01, 0x00, 0x00, 0x00}, cleanPos)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	w2, err := Open(dir, gate)
	require.NoError(t, err)
	defer w2.Close()

	assert.Equal(t, cleanPos, w2.WritePos())
	assert.Equal(t, uint64(5), w2.NextSeq())

	for i := range uint64(5) {
		hdr, payload, err := w2.ReadRecord(i)
		require.NoError(t, err)
		assert.True(t, hdr.Has(WalFlagAppended))
		assert.Equal(t, []byte{byte('a' + i)}, payload)
	}
}
