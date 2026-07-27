package wal

import (
	"testing"

	"espx/pkg/dedupkey"
	"espx/pkg/iogate"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWAL_ForwardClaimAndRemoteAck(t *testing.T) {
	dir := t.TempDir()
	gate := iogate.NewDiskWriteGate(iogate.Config{AppendCapacity: 8, GroupCommitRecords: 1})
	w, err := Open(dir, gate)
	require.NoError(t, err)
	defer w.Close()

	_, err = w.Append([]byte("uplink"))
	require.NoError(t, err)

	var buf [MaxPayloadSize + 64]byte
	_, err = w.ProcessPendingKeyGen(1, func(seq uint64, p []byte) ([32]byte, error) {
		id := dedupkey.FactorU(dedupkey.WriteCanonicalProxyBatchPayload(buf[:0], seq, p))
		var out [32]byte
		copy(out[:], id[:])
		return out, nil
	})
	require.NoError(t, err)

	claimed, err := w.TryClaimForward(0)
	require.NoError(t, err)
	require.True(t, claimed)

	claimed, err = w.TryClaimForward(0)
	require.NoError(t, err)
	assert.False(t, claimed)

	require.NoError(t, w.MarkRemoteAcked(0))

	hdr, _, err := w.ReadRecord(0)
	require.NoError(t, err)
	assert.True(t, hdr.Has(WalFlagForwardClaimed))
	assert.True(t, hdr.Has(WalFlagRemoteAcked))
}
