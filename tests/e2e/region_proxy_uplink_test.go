package e2e_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"espx/internal/config"
	"espx/internal/database"
	"espx/internal/ingestion"
	"espx/internal/controlplane"
	"espx/pkg/dedupkey"
	"espx/pkg/iogate"
	"espx/pkg/regionproxy/keygen"
	"espx/pkg/regionproxy/opkey"
	"espx/pkg/regionproxy/uplink"
	"espx/pkg/regionproxy/wal"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestE2E_RegionProxyUplink(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping region-proxy uplink e2e")
	}

	ctx := context.Background()
	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()

	_, err := pool.Exec(ctx, `
		INSERT INTO regions (code, name, active) VALUES (1, 'us-east', TRUE)
		ON CONFLICT (code) DO NOTHING`)
	require.NoError(t, err)

	cfg := &config.Config{
		MultiRegionEnabled: true,
		RegionCode:         0,
		AdminAPIKey:        "e2e-uplink-key",
	}
	svc := controlplane.NewService(pool, nil, ingestion.NewStaticSlotSharder(1), cfg)
	t.Cleanup(func() { svc.Close() })
	handler := controlplane.NewHandler(svc, cfg, nil, nil, nil, nil)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	globalSrv := httptest.NewServer(mux)
	defer globalSrv.Close()

	dataDir := t.TempDir()
	gate := iogate.NewDiskWriteGate(iogate.Config{AppendCapacity: 8, GroupCommitRecords: 1})
	segment, err := wal.Open(dataDir, gate)
	require.NoError(t, err)
	defer segment.Close()

	payload := []byte(`{"e2e":"uplink-batch"}`)
	_, err = segment.Append(payload)
	require.NoError(t, err)

	kg := keygen.New(segment, keygen.Config{RegionCode: 1, NodeID: "e2e-proxy", PollInterval: time.Millisecond, BatchSize: 8})
	kg.Start()
	defer kg.Stop()

	require.NoError(t, segment.WaitKeyGenReady(ctx, time.Millisecond))

	opPool := opkey.New(segment, opkey.Config{NodeID: "e2e-proxy", QueueSize: 64, Watermark: 1000, PollInterval: time.Millisecond})
	opPool.Start()
	defer opPool.Stop()

	for opPool.Enqueued() < 1 {
		time.Sleep(time.Millisecond)
	}
	slot, ok := opPool.Dequeue()
	require.True(t, ok)

	worker := uplink.New(segment, opPool, uplink.Config{
		RegionCode:  1,
		NodeID:      "e2e-proxy",
		GlobalURL:   globalSrv.URL + "/api/v1/region/ingest/batch",
		APIKey:      string(cfg.AdminAPIKey),
		HTTPTimeout: 5 * time.Second,
	})
	require.NoError(t, worker.ForwardOnce(slot))
	opPool.Release(slot)

	hdr, _, err := segment.ReadRecord(0)
	require.NoError(t, err)
	assert.True(t, hdr.Has(wal.WalFlagForwardClaimed))
	assert.True(t, hdr.Has(wal.WalFlagRemoteAcked))

	var proposalCount int
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM dedup_key_proposals`).Scan(&proposalCount))
	assert.Equal(t, 1, proposalCount)

	var buf [256]byte
	expected := dedupkey.FactorU(dedupkey.WriteCanonicalProxyBatchPayload(buf[:0], 0, payload))
	assert.Equal(t, expected, dedupkey.FactorU(dedupkey.WriteCanonicalProxyBatchPayload(buf[:0], 0, payload)))
}
