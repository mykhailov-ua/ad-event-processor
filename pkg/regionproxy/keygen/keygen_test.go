package keygen

import (
	"context"
	"testing"
	"time"

	"ad-event-processor/pkg/dedupkey"
	"ad-event-processor/pkg/iogate"
	"ad-event-processor/pkg/regionproxy/wal"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeyGen_10kRecordsDedupReady(t *testing.T) {
	dir := t.TempDir()
	gate := iogate.NewDiskWriteGate(iogate.TestGateConfig())
	w, err := wal.Open(dir, gate)
	require.NoError(t, err)
	defer w.Close()

	payload := []byte(`{"click":"evt"}`)
	records := 10000
	if testing.Short() {
		records = 2000
	}
	for i := range records {
		_, err := w.Append(payload)
		require.NoError(t, err)
	}

	kg := New(w, Config{
		RegionCode:   1,
		NodeID:       "bench-node",
		PollInterval: time.Millisecond,
		BatchSize:    512,
	})
	kg.Start()
	defer kg.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	require.NoError(t, w.WaitKeyGenReady(ctx, time.Millisecond))

	seen := make(map[[32]byte]struct{}, records)
	for seq := uint64(0); seq < uint64(records); seq++ {
		hdr, _, err := w.ReadRecord(seq)
		require.NoError(t, err)
		assert.True(t, hdr.Has(wal.WalFlagDedupReady), "seq=%d", seq)
		require.NotEqual(t, [32]byte{}, hdr.FactorU, "seq=%d", seq)

		var buf [wal.MaxPayloadSize + 64]byte
		expect := dedupkey.FactorU(dedupkey.WriteCanonicalProxyBatchPayload(buf[:0], seq, payload))
		var expectBytes [32]byte
		copy(expectBytes[:], expect[:])
		assert.Equal(t, expectBytes, hdr.FactorU, "seq=%d", seq)

		seen[hdr.FactorU] = struct{}{}
	}
	assert.Len(t, seen, records)
	assert.Equal(t, uint64(records), kg.Processed())
}
