package controlplane

import (
	"espx/pkg/faultproof"

	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"espx/pkg/dedupkey"
	"espx/pkg/iogate"
	"espx/pkg/regionproxy/wal"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFault_ProxyKeyGenCPUThrottle(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	gate := iogate.NewDiskWriteGate(iogate.Config{AppendCapacity: 8, GroupCommitRecords: 1})
	w, err := wal.Open(dir, gate)
	require.NoError(t, err)
	defer w.Close()

	const n = 500
	payload := []byte(`{"region":"us-east"}`)
	for i := 0; i < n; i++ {
		_, err := w.Append(payload)
		require.NoError(t, err)
	}

	var buf [wal.MaxPayloadSize + 64]byte
	throttle := func(seq uint64, p []byte) ([32]byte, error) {
		time.Sleep(100 * time.Microsecond)
		canon := dedupkey.WriteCanonicalProxyBatchPayload(buf[:0], seq, p)
		id := dedupkey.FactorU(canon)
		var out [32]byte
		copy(out[:], id[:])
		return out, nil
	}

	processed := 0
	for processed < n {
		gate.SetDegraded(processed%50 == 25)
		batch, err := w.ProcessPendingKeyGen(32, throttle)
		processed += batch
		if err != nil {
			require.True(t, errors.Is(err, iogate.ErrShed) || batch > 0)
		}
		if batch == 0 {
			time.Sleep(time.Millisecond)
		}
	}
	gate.SetDegraded(false)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	require.NoError(t, w.WaitKeyGenReady(ctx, time.Millisecond))

	seen := make(map[[32]byte]uint64, n)
	factors := make([][32]byte, n)
	for seq := uint64(0); seq < n; seq++ {
		hdr, _, err := w.ReadRecord(seq)
		require.NoError(t, err)
		require.True(t, hdr.Has(wal.WalFlagDedupReady))
		if prev, ok := seen[hdr.FactorU]; ok {
			t.Fatalf("duplicate factor_u at seq %d and %d", prev, seq)
		}
		seen[hdr.FactorU] = seq
		factors[seq] = hdr.FactorU
	}
	assert.Len(t, seen, n)

	for round := 0; round < 2; round++ {
		_, err := w.ProcessPendingKeyGen(n, throttle)
		require.NoError(t, err)
	}
	for seq := uint64(0); seq < n; seq++ {
		hdr, _, err := w.ReadRecord(seq)
		require.NoError(t, err)
		assert.Equal(t, factors[seq], hdr.FactorU, "seq=%d", seq)
	}

	faultproof.Log(t, "mr_keygen_throttle", map[string]string{
		"subsystem":     "region_proxy_keygen",
		"proposal_rows": "1",
		"records":       strconv.Itoa(n),
		"baseline_ok":   "true",
	})
}
