package management

import (
	"context"
	"strconv"
	"testing"

	"espx/internal/config"
	"espx/internal/database"
	"espx/internal/dedup"
	"espx/internal/ingestion"
	db "espx/internal/ingestion/sqlc"
	"espx/pkg/dedupkey"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// TestChaos_RegionProxyQuorumBook_1of3NoExecute is CH-MR-09: no apply until 2-of-3 book ACKs.
func TestChaos_RegionProxyQuorumBook_1of3NoExecute(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test")
	}

	ctx := context.Background()
	pool, cleanup := database.SetupTestDB(t)
	t.Cleanup(cleanup)

	_, err := pool.Exec(ctx, `
		INSERT INTO regions (code, name, active) VALUES (1, 'us-east', TRUE)
		ON CONFLICT (code) DO NOTHING`)
	require.NoError(t, err)

	cfg := &config.Config{MultiRegionEnabled: true, RegionCode: 0, NodeID: "global-quorum"}
	svc := newBareService(t, pool, nil, cfg)
	worker := NewOperationLeaseWorker(svc)

	opID := uuid.New()
	replicaSetID := uuid.New()
	scope := dedup.NewAdapter(pool, 1, 1).RegionScope(dedupkey.RelaySourceID(1), 51, 51)
	_, err = worker.Book(ctx, OperationLeaseBookRequest{
		OpID:         opID,
		RegionCode:   1,
		Role:         "management",
		ReplicaSetID: replicaSetID,
		Attempt:      1,
		FactorU:      uuid.New(),
		Scope:        scope,
		ReplicaNodes: []string{"proxy-a", "proxy-b", "proxy-c"},
		BookAckNodes: []string{"proxy-a"},
	})
	require.ErrorIs(t, err, ErrLeaseQuorumNotMet)

	var proposalCount int
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM dedup_key_proposals`).Scan(&proposalCount))
	require.Equal(t, 0, proposalCount)

	err = worker.ExecuteOp(ctx, opID, func(ctx context.Context, _ db.OperationLease, _ dedup.ClaimResult) error {
		t.Fatal("execute must not run without quorum")
		return nil
	})
	require.NoError(t, err)
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM dedup_key_proposals`).Scan(&proposalCount))
	require.Equal(t, 0, proposalCount)

	status, err := worker.AckBook(ctx, opID, "proxy-b")
	require.NoError(t, err)
	require.True(t, status.QuorumMet)
	require.Equal(t, int32(2), status.AckCount)

	err = worker.ExecuteOp(ctx, opID, func(ctx context.Context, _ db.OperationLease, _ dedup.ClaimResult) error {
		return nil
	})
	require.NoError(t, err)

	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM dedup_key_proposals`).Scan(&proposalCount))
	require.Equal(t, 1, proposalCount)

	lease, err := db.New(pool).GetOperationLease(ctx, ingestion.ToUUID(opID))
	require.NoError(t, err)
	require.Equal(t, string(LeaseStateCompleted), lease.LeaseState)

	logChaosProof(t, "mr_quorum_book", map[string]string{
		"subsystem":     "region_proxy_quorum",
		"op_id":         opID.String(),
		"quorum_acks":   "2",
		"client_ack":    "true",
		"proposal_rows": strconv.Itoa(proposalCount),
		"baseline_ok":   "true",
	})
}

// TestChaos_RegionProxyQuorumBook_Kill2of3Simulated proves WAL/book stays unacked with only 1-of-3 ACKs.
func TestChaos_RegionProxyQuorumBook_Kill2of3Simulated(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test")
	}

	ctx := context.Background()
	pool, cleanup := database.SetupTestDB(t)
	t.Cleanup(cleanup)

	cfg := &config.Config{MultiRegionEnabled: true, RegionCode: 1, NodeID: "proxy-survivor"}
	svc := newBareService(t, pool, nil, cfg)
	worker := NewOperationLeaseWorker(svc)

	opID := uuid.New()
	scope := dedup.NewAdapter(pool, 1, 1).RegionScope(dedupkey.RelaySourceID(1), 52, 52)
	bookRes, err := worker.Book(ctx, OperationLeaseBookRequest{
		OpID:         opID,
		RegionCode:   1,
		Role:         "management",
		ReplicaSetID: uuid.New(),
		Attempt:      1,
		FactorU:      uuid.New(),
		Scope:        scope,
		ReplicaNodes: []string{"proxy-a", "proxy-b", "proxy-c"},
		BookAckNodes: []string{"proxy-a"},
	})
	require.ErrorIs(t, err, ErrLeaseQuorumNotMet)
	require.False(t, bookRes.QuorumMet)
	require.Equal(t, int32(1), bookRes.AckCount)

	var proposalCount int
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM dedup_key_proposals`).Scan(&proposalCount))
	require.Equal(t, 0, proposalCount)

	logChaosProof(t, "mr_quorum_book_partition", map[string]string{
		"subsystem":   "region_proxy_quorum",
		"op_id":       opID.String(),
		"quorum_acks": "1",
		"client_ack":  "false",
		"baseline_ok": "true",
	})
}
