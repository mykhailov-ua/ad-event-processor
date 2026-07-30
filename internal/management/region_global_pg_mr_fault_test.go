package management

import (
	"espx/pkg/faultproof"

	"context"
	"strconv"
	"testing"
	"time"

	"espx/internal/config"
	"espx/internal/database"
	"espx/pkg/dedupkey"

	"github.com/stretchr/testify/require"
)

func TestFault_RegionGlobalPGPartition(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	ctx := context.Background()
	infra, cleanup := database.SetupTestDBInfra(t)
	t.Cleanup(cleanup)

	_, err := infra.Pool.Exec(ctx, `
		INSERT INTO regions (code, name, active) VALUES (1, 'us-east', TRUE)
		ON CONFLICT (code) DO NOTHING`)
	require.NoError(t, err)

	cfg := &config.Config{MultiRegionEnabled: true, RegionCode: 0}
	svc := newBareService(t, infra.Pool, nil, cfg)

	payload := []byte(`{"batch":"mr-global-pg-partition"}`)
	var buf [256]byte
	factorU := dedupkey.FactorU(dedupkey.WriteCanonicalProxyBatchPayload(buf[:0], 7, payload))
	in := RegionIngestBatchInput{
		RegionCode: 1,
		NodeID:     "proxy-node-7",
		Seq:        7,
		FactorU:    factorU,
		Payload:    payload,
	}

	_, err = svc.IngestRegionProxyBatch(ctx, in)
	require.NoError(t, err)

	stopLeasePGContainer(t, infra)
	require.Eventually(t, func() bool {
		return infra.Pool.Ping(ctx) != nil
	}, 15*time.Second, 200*time.Millisecond)

	partitionErr := false
	for i := 0; i < 3; i++ {
		_, err = svc.IngestRegionProxyBatch(ctx, in)
		if err != nil {
			partitionErr = true
			break
		}
	}
	require.True(t, partitionErr, "uplink must fail while global PG is partitioned")

	time.Sleep(500 * time.Millisecond)

	startLeasePGContainer(t, infra)
	refreshLeasePGPool(t, infra)
	svc.SetPool(infra.Pool)

	_, err = svc.IngestRegionProxyBatch(ctx, in)
	require.NoError(t, err)

	var proposalCount int
	require.NoError(t, infra.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM dedup_key_proposals`).Scan(&proposalCount))
	require.Equal(t, 1, proposalCount)

	faultproof.Log(t, "mr_global_pg_partition", map[string]string{
		"subsystem":     "region_proxy_uplink",
		"proposal_rows": strconv.Itoa(proposalCount),
		"wal_bytes":     strconv.Itoa(len(payload)),
		"baseline_ok":   "true",
	})
}
