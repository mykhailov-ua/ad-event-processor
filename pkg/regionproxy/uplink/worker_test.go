package uplink

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/pkg/dedupkey"
	"github.com/bidshard/ad-event-processor/pkg/iogate"
	"github.com/bidshard/ad-event-processor/pkg/regionproxy/opkey"
	"github.com/bidshard/ad-event-processor/pkg/regionproxy/wal"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func prepareKeyedWAL(t *testing.T) *wal.WAL {
	dir := t.TempDir()
	gate := iogate.NewDiskWriteGate(iogate.TestGateConfig())
	w, err := wal.Open(dir, gate)
	require.NoError(t, err)
	t.Cleanup(func() { _ = w.Close() })

	_, err = w.Append([]byte("iso06-uplink"))
	require.NoError(t, err)

	var buf [wal.MaxPayloadSize + 64]byte
	_, err = w.ProcessPendingKeyGen(1, func(seq uint64, p []byte) ([32]byte, error) {
		id := dedupkey.FactorU(dedupkey.WriteCanonicalProxyBatchPayload(buf[:0], seq, p))
		var out [32]byte
		copy(out[:], id[:])
		return out, nil
	})
	require.NoError(t, err)
	return w
}

func TestWorker_forwardRetriesAfterHTTPFailure(t *testing.T) {
	t.Parallel()

	w := prepareKeyedWAL(t)
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, _ *http.Request) {
		if attempts.Add(1) < 3 {
			resp.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		resp.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	worker := New(w, nil, Config{
		RegionCode:          1,
		NodeID:              "iso06",
		GlobalURL:           srv.URL,
		ForwardMaxAttempts:  3,
		ForwardRetryBackoff: time.Millisecond,
		HTTPTimeout:         2 * time.Second,
	})

	slot := &opkey.Slot{Seq: 0}
	require.NoError(t, worker.ForwardOnce(slot), "harness=region_proxy_uplink_worker: retries must ack after transient 503")

	assert.GreaterOrEqual(t, int(attempts.Load()), 3)
	assert.Equal(t, uint64(1), worker.Acked())
	assert.GreaterOrEqual(t, worker.Forwarded(), uint64(2))

	hdr, _, err := w.ReadRecord(0)
	require.NoError(t, err)
	assert.True(t, hdr.Has(wal.WalFlagForwardClaimed))
	assert.True(t, hdr.Has(wal.WalFlagRemoteAcked))
}

func TestWorker_forwardUnclaimsOnPersistentFailure(t *testing.T) {
	t.Parallel()

	w := prepareKeyedWAL(t)
	srv := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, _ *http.Request) {
		resp.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	worker := New(w, nil, Config{
		RegionCode:          1,
		NodeID:              "iso06",
		GlobalURL:           srv.URL,
		ForwardMaxAttempts:  2,
		ForwardRetryBackoff: time.Millisecond,
		HTTPTimeout:         2 * time.Second,
	})

	slot := &opkey.Slot{Seq: 0}
	require.Error(t, worker.ForwardOnce(slot))
	assert.Equal(t, uint64(0), worker.Acked())

	hdr, _, err := w.ReadRecord(0)
	require.NoError(t, err)
	assert.False(t, hdr.Has(wal.WalFlagForwardClaimed), "claim must be cleared so the next poll can retry")
	assert.False(t, hdr.Has(wal.WalFlagRemoteAcked))

	claimed, err := w.TryClaimForward(0)
	require.NoError(t, err)
	assert.True(t, claimed, "unclaimed WAL record must accept a new forward claim")
}
