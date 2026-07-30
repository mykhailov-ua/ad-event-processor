package management

import (
	"espx/pkg/faultproof"

	"context"
	"strconv"
	"testing"
	"time"

	"espx/internal/config"
	"espx/internal/database"
	"espx/internal/dedup"
	"espx/internal/ingestion"
	db "espx/internal/ingestion/sqlc"
	"espx/pkg/dedupkey"
	"espx/pkg/regionproxy/opkey"
	"espx/pkg/regionproxy/quorum"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestFault_RegionProxyQuorumBook_1of3NoExecute(t *testing.T) {
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

	faultproof.Log(t, "mr_quorum_book", map[string]string{
		"subsystem":     "region_proxy_quorum",
		"op_id":         opID.String(),
		"quorum_acks":   "2",
		"client_ack":    "true",
		"proposal_rows": strconv.Itoa(proposalCount),
		"baseline_ok":   "true",
	})
}

func TestFault_RegionProxyQuorumBook_Kill2of3Simulated(t *testing.T) {
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

	faultproof.Log(t, "mr_quorum_book_partition", map[string]string{
		"subsystem":   "region_proxy_quorum",
		"op_id":       opID.String(),
		"quorum_acks": "1",
		"client_ack":  "false",
		"baseline_ok": "true",
	})
}

func TestFault_QuorumBook_WithPGDown(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	ctx := context.Background()
	infra, cleanup := database.SetupTestDBInfra(t)
	t.Cleanup(cleanup)
	rdb, cleanupRedis := database.SetupTestRedis(t)
	t.Cleanup(cleanupRedis)

	cfg := &config.Config{MultiRegionEnabled: true, RegionCode: 1, NodeID: "proxy-a"}
	svc := newBareService(t, infra.Pool, []redis.UniversalClient{rdb}, cfg)
	worker := NewOperationLeaseWorker(svc)

	opID := uuid.New()
	replicas := []string{"proxy-a", "proxy-b", "proxy-c"}
	scope := dedup.NewAdapter(infra.Pool, 1, 1).RegionScope(dedupkey.RelaySourceID(1), 77, 77)

	stopLeasePGContainer(t, infra)
	require.Eventually(t, func() bool {
		return infra.Pool.Ping(ctx) != nil
	}, 15*time.Second, 200*time.Millisecond)

	bookRes, err := worker.Book(ctx, OperationLeaseBookRequest{
		OpID:         opID,
		RegionCode:   1,
		Role:         "region-proxy",
		ReplicaSetID: uuid.New(),
		Attempt:      1,
		FactorU:      uuid.New(),
		Scope:        scope,
		ReplicaNodes: replicas,
		BookAckNodes: []string{"proxy-a"},
	})
	require.ErrorIs(t, err, ErrLeaseQuorumNotMet)
	require.False(t, bookRes.QuorumMet)
	require.Equal(t, int32(1), bookRes.AckCount)

	var slot opkey.Slot
	slot.Seq = 99
	slot.SetDerivedForTest()
	copy(slot.OpID[:], opID[:])
	committer := opkey.NewBatchCommitter(rdb, "proxy-a", replicas)
	ready, err := committer.PrepareForward(ctx, &slot)
	require.NoError(t, err)
	require.False(t, ready)
	require.Equal(t, uint64(0), committer.Committed())

	status, err := worker.AckBook(ctx, opID, "proxy-b")
	require.NoError(t, err)
	require.True(t, status.QuorumMet)
	require.Equal(t, int32(2), status.AckCount)

	ready, err = committer.PrepareForward(ctx, &slot)
	require.NoError(t, err)
	require.True(t, ready)
	committer.Complete(ctx, &slot)
	require.Equal(t, uint64(1), committer.Committed())

	st, err := quorum.ReadStatus(ctx, rdb, opIDBytes(opID), len(replicas))
	require.NoError(t, err)
	require.Equal(t, quorum.StateCompleted, st.State)

	faultproof.Log(t, "mr_quorum_book_pg_down", map[string]string{
		"subsystem":     "region_proxy_quorum",
		"op_id":         opID.String(),
		"quorum_acks":   "2",
		"client_ack":    "true",
		"opkey_commits": "1",
		"baseline_ok":   "true",
	})
}
