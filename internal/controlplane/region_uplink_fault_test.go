package controlplane

import (
	"github.com/bidshard/ad-event-processor/pkg/faultproof"

	"context"
	"strconv"
	"testing"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/database"
	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/bidshard/ad-event-processor/pkg/dedupkey"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestFault_RegionUplinkDedup(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test")
	}

	ctx := context.Background()
	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()

	_, err := pool.Exec(ctx, `
		INSERT INTO regions (code, name, active) VALUES (1, 'us-east', TRUE)
		ON CONFLICT (code) DO NOTHING`)
	require.NoError(t, err)

	cfg := &config.Config{MultiRegionEnabled: true, RegionCode: 0}
	svc := newBareService(t, pool, nil, cfg)

	payload := []byte(`{"batch":"mr-uplink"}`)
	var buf [256]byte
	factorU := dedupkey.FactorU(dedupkey.WriteCanonicalProxyBatchPayload(buf[:0], 0, payload))
	in := RegionIngestBatchInput{
		RegionCode: 1,
		NodeID:     "proxy-node-1",
		Seq:        0,
		FactorU:    factorU,
		Payload:    payload,
	}

	for i := 0; i < 3; i++ {
		_, err := svc.IngestRegionProxyBatch(ctx, in)
		require.NoError(t, err)
	}

	var proposalCount int
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM dedup_key_proposals`).Scan(&proposalCount))
	require.Equal(t, 1, proposalCount)

	opID := ProxyBatchOpID(1, "proxy-node-1", 0, uuid.Nil)
	var leaseState string
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT lease_state FROM operation_leases WHERE op_id = $1`, domain.ToUUID(opID)).Scan(&leaseState))
	require.Equal(t, string(LeaseStateCompleted), leaseState)

	faultproof.Log(t, "mr_uplink_dedup", map[string]string{
		"subsystem":     "region_proxy_uplink",
		"region_code":   "1",
		"proposal_rows": strconv.Itoa(proposalCount),
		"replays":       "3",
		"lease_state":   leaseState,
		"baseline_ok":   "true",
	})
}

func TestRegionIngestBatch_FactorMismatchRejected(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test")
	}

	ctx := context.Background()
	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()

	cfg := &config.Config{MultiRegionEnabled: true, RegionCode: 0}
	svc := newBareService(t, pool, nil, cfg)

	_, err := svc.IngestRegionProxyBatch(ctx, RegionIngestBatchInput{
		RegionCode: 1,
		NodeID:     "proxy-node-1",
		Seq:        1,
		FactorU:    uuid.New(),
		Payload:    []byte("payload"),
	})
	require.Error(t, err)
}
