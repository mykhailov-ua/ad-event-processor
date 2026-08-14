package opkey

import (
	"context"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/pkg/dedupkey"
	"github.com/bidshard/ad-event-processor/pkg/iogate"
	"github.com/bidshard/ad-event-processor/pkg/regionproxy/wal"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPool_DrainsWALDedupReady(t *testing.T) {
	dir := t.TempDir()
	gate := iogate.NewDiskWriteGate(iogate.TestGateConfig())
	w, err := wal.Open(dir, gate)
	require.NoError(t, err)
	defer w.Close()

	payload := []byte("batch")
	for range 100 {
		_, err := w.Append(payload)
		require.NoError(t, err)
	}

	var buf [wal.MaxPayloadSize + 64]byte
	_, err = w.ProcessPendingKeyGen(100, func(seq uint64, p []byte) ([32]byte, error) {
		canon := dedupkey.WriteCanonicalProxyBatchPayload(buf[:0], seq, p)
		id := dedupkey.FactorU(canon)
		var out [32]byte
		copy(out[:], id[:])
		return out, nil
	})
	require.NoError(t, err)

	pool := New(w, Config{NodeID: "drain-test", QueueSize: 256, Watermark: 1000, PollInterval: time.Millisecond})
	pool.Start()
	defer pool.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for pool.Enqueued() < 100 {
		select {
		case <-ctx.Done():
			t.Fatalf("timeout: enqueued=%d", pool.Enqueued())
		default:
			time.Sleep(time.Millisecond)
		}
	}

	seen := make(map[uint64]struct{}, 100)
	for range 100 {
		slot, ok := pool.Dequeue()
		require.True(t, ok)
		assert.True(t, slot.Has(OpKeyFlagDerived))
		assert.NotEqual(t, [16]byte{}, slot.OpID)
		seen[slot.Seq] = struct{}{}
		pool.Release(slot)
	}
	assert.Len(t, seen, 100)
}
