package wal

import (
	"os"
	"testing"

	"espx/pkg/iogate"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWAL_ChaosSIGKILLReplayIdempotent(t *testing.T) {
	dir := t.TempDir()
	gate := iogate.NewDiskWriteGate(iogate.Config{AppendCapacity: 8, GroupCommitRecords: 1})

	appendRecords := func() uint64 {
		w, err := Open(dir, gate)
		require.NoError(t, err)
		for i := 0; i < 10; i++ {
			_, err := w.Append([]byte{byte('a' + i)})
			require.NoError(t, err)
		}
		seq := w.NextSeq()
		require.NoError(t, w.Close())
		return seq
	}

	wantSeq := appendRecords()

	w2, err := Open(dir, gate)
	require.NoError(t, err)
	defer w2.Close()
	assert.Equal(t, wantSeq, w2.NextSeq())

	for i := uint64(0); i < wantSeq; i++ {
		hdr, payload, err := w2.ReadRecord(i)
		require.NoError(t, err)
		assert.True(t, hdr.Has(WalFlagAppended))
		assert.Equal(t, []byte{byte('a' + i)}, payload)
	}

	// Simulate SIGKILL: reopen without graceful close still replays same offsets.
	w3, err := Open(dir, gate)
	require.NoError(t, err)
	assert.Equal(t, wantSeq, w3.NextSeq())
	require.NoError(t, w3.Close())

	// Torn tail after crash is truncated and replay stays idempotent.
	path := dir + "/" + walSegmentFile
	fi, err := os.Stat(path)
	require.NoError(t, err)
	f, err := os.OpenFile(path, os.O_RDWR, 0644)
	require.NoError(t, err)
	_, err = f.WriteAt([]byte{0xFF, 0xFF, 0xFF, 0xFF}, fi.Size()-4)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	w4, err := Open(dir, gate)
	require.NoError(t, err)
	defer w4.Close()
	assert.Equal(t, wantSeq, w4.NextSeq())
}
