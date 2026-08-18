package ingestion

import (
	"context"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/pkg/dedupkey"
	"github.com/bidshard/ad-event-processor/pkg/iogate"
	rpclient "github.com/bidshard/ad-event-processor/pkg/regionproxy/client"
	rserver "github.com/bidshard/ad-event-processor/pkg/regionproxy/server"
	"github.com/bidshard/ad-event-processor/pkg/regionproxy/wal"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type regionProxySpendSyncAdapter struct {
	client *rpclient.Client
}

func (a regionProxySpendSyncAdapter) ProduceSpendSyncPayload(payload []byte) (SpendSyncBatchResult, error) {
	res, err := a.client.ProduceSpendSyncPayload(payload)
	return SpendSyncBatchResult{Committed: res.Committed}, err
}

func TestSpendSyncProducer_FlushToRegionProxy(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	ctx := context.Background()
	dataDir := t.TempDir()
	gate := iogate.NewDiskWriteGate(iogate.Config{AppendCapacity: 16, GroupCommitRecords: 1})
	srv, err := rserver.NewServer("127.0.0.1:0", dataDir, gate)
	require.NoError(t, err)
	require.NoError(t, srv.Start())
	defer srv.Stop()
	time.Sleep(150 * time.Millisecond)

	client := rpclient.New(rpclient.Config{Addr: srv.Addr(), Timeout: 2 * time.Second})
	defer func() { _ = client.Close() }()

	producer := NewSpendSyncProducer(regionProxySpendSyncAdapter{client: client}, 2)
	campID := uuid.New()

	for i := range 2 {
		entry := PendingRollup{
			AmountMicro: 1_000,
			TxID:        "sync-txn-" + string(rune('a'+i)),
			IDStr:       campID.String(),
		}
		require.NoError(t, producer.EnqueueRollup(ctx, nil, campID, entry))
	}

	assert.Equal(t, 0, producer.PendingCount())

	segment, err := wal.Open(dataDir, gate)
	require.NoError(t, err)
	defer func() { _ = segment.Close() }()
	assert.Equal(t, uint64(1), segment.NextSeq())

	hdr, payload, err := segment.ReadRecord(0)
	require.NoError(t, err)
	assert.True(t, hdr.Has(wal.WalFlagAppended))
	assert.True(t, dedupkey.IsSpendSyncPayload(payload))

	txns, err := dedupkey.DecodeSpendSyncPayload(payload)
	require.NoError(t, err)
	require.Len(t, txns, 2)
	assert.Equal(t, campID, txns[0].CampaignID)
}
