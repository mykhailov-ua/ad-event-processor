package controlplane

import (
	"context"
	"testing"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/database"
	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/bidshard/ad-event-processor/pkg/dedupkey"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestProxyUplink_IngestUsesOperationLease(t *testing.T) {
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

	payload := []byte(`{"batch":"lease-uplink"}`)
	var buf [256]byte
	factorU := dedupkey.FactorU(dedupkey.WriteCanonicalProxyBatchPayload(buf[:0], 42, payload))
	opID := uuid.New()
	in := RegionIngestBatchInput{
		RegionCode: 1,
		NodeID:     "proxy-node-1",
		Seq:        42,
		FactorU:    factorU,
		Payload:    payload,
		OpID:       opID,
	}

	result, err := svc.IngestRegionProxyBatch(ctx, in)
	require.NoError(t, err)
	require.NotEmpty(t, result.DedupKey)

	var leaseState string
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT lease_state FROM operation_leases WHERE op_id = $1`, domain.ToUUID(opID)).Scan(&leaseState))
	require.Equal(t, string(LeaseStateCompleted), leaseState)
}

func TestRelayDeliveryOpID_Deterministic(t *testing.T) {
	t.Parallel()
	a := RelayDeliveryOpID(1, 99)
	b := RelayDeliveryOpID(1, 99)
	c := RelayDeliveryOpID(1, 100)
	require.Equal(t, a, b)
	require.NotEqual(t, a, c)
}
