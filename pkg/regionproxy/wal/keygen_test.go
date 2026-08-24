package wal

import (
	"testing"

	"ad-event-processor/pkg/dedupkey"
	"ad-event-processor/pkg/iogate"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWAL_ProcessPendingKeyGen(t *testing.T) {
	dir := t.TempDir()
	gate := iogate.NewDiskWriteGate(iogate.TestGateConfig())
	w, err := Open(dir, gate)
	require.NoError(t, err)
	defer w.Close()

	_, err = w.Append([]byte("alpha"))
	require.NoError(t, err)
	_, err = w.Append([]byte("beta"))
	require.NoError(t, err)

	var buf [MaxPayloadSize + 64]byte
	derive := func(seq uint64, payload []byte) ([32]byte, error) {
		canon := dedupkey.WriteCanonicalProxyBatchPayload(buf[:0], seq, payload)
		id := dedupkey.FactorU(canon)
		var out [32]byte
		copy(out[:], id[:])
		return out, nil
	}

	n, err := w.ProcessPendingKeyGen(8, derive)
	require.NoError(t, err)
	assert.Equal(t, 2, n)
	assert.Equal(t, int64(0), w.KeyGenQueueDepth())

	hdr, _, err := w.ReadRecord(0)
	require.NoError(t, err)
	assert.True(t, hdr.Has(WalFlagDedupReady))
	assert.NotEqual(t, [32]byte{}, hdr.FactorU)

	n, err = w.ProcessPendingKeyGen(8, derive)
	require.NoError(t, err)
	assert.Equal(t, 0, n)
}

func TestWAL_KeyGenQueueDepth(t *testing.T) {
	dir := t.TempDir()
	gate := iogate.NewDiskWriteGate(iogate.TestGateConfig())
	w, err := Open(dir, gate)
	require.NoError(t, err)
	defer w.Close()

	_, err = w.Append([]byte("x"))
	require.NoError(t, err)
	assert.Equal(t, int64(1), w.KeyGenQueueDepth())
}
