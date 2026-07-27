package wal

import (
	"testing"

	"espx/pkg/iogate"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWAL_AppendRecoverRoundTrip(t *testing.T) {
	dir := t.TempDir()
	gate := iogate.NewDiskWriteGate(iogate.Config{AppendCapacity: 8, GroupCommitRecords: 1})

	w, err := Open(dir, gate)
	require.NoError(t, err)
	defer w.Close()

	seq, err := w.Append([]byte("alpha"))
	require.NoError(t, err)
	assert.Equal(t, uint64(0), seq)

	seq, err = w.Append([]byte("beta"))
	require.NoError(t, err)
	assert.Equal(t, uint64(1), seq)

	require.NoError(t, w.Close())

	w2, err := Open(dir, gate)
	require.NoError(t, err)
	defer w2.Close()

	assert.Equal(t, uint64(2), w2.NextSeq())

	hdr, payload, err := w2.ReadRecord(0)
	require.NoError(t, err)
	assert.True(t, hdr.Has(WalFlagAppended))
	assert.Equal(t, []byte("alpha"), payload)

	hdr, payload, err = w2.ReadRecord(1)
	require.NoError(t, err)
	assert.True(t, hdr.Has(WalFlagAppended))
	assert.Equal(t, []byte("beta"), payload)
}
