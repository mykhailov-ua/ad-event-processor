package controlplane

import (
	"context"
	"sync"
	"testing"
	"time"

	"espx/internal/config"
	"espx/internal/database"
	"espx/internal/dedup"
	"espx/internal/domain"
	db "espx/internal/domain/db"
	"espx/pkg/dedupkey"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type stubOpKeyGate struct {
	shed bool
}

func (g stubOpKeyGate) ShouldShed() bool { return g.shed }

func TestOperationLeaseConditions(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test")
	}

	cases := []struct {
		name string
		run  func(t *testing.T)
	}{
		{name: "C1_authoritative_pg_deadline", run: testConditionC1AuthoritativeDeadline},
		{name: "C2_fencing_epoch_file", run: testConditionC2FencingEpoch},
		{name: "C3_quorum_book_ack", run: testConditionC3QuorumBook},
		{name: "C4_deadline_from_pg_now", run: testConditionC4DeadlineFromPG},
		{name: "C5_attempt_in_d3_scope", run: testConditionC5AttemptInScope},
		{name: "C6_renew_max_budget", run: testConditionC6RenewMax},
		{name: "C7_opkey_pool_shed", run: testConditionC7OpKeyShed},
		{name: "C8_janitor_leader_lock", run: testConditionC8JanitorLeader},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}

func testConditionC1AuthoritativeDeadline(t *testing.T) {
	ctx := context.Background()
	pool, cleanup := database.SetupTestDB(t)
	t.Cleanup(cleanup)

	cfg := &config.Config{MultiRegionEnabled: true, RegionCode: 1, NodeID: "c1-node", OpLeaseTimeoutSec: 30}
	svc := newBareService(t, pool, nil, cfg)
	worker := NewOperationLeaseWorker(svc)

	opID := uuid.New()
	scope := dedup.NewAdapter(pool, 1, 1).RegionScope(dedupkey.RelaySourceID(1), 5, 5)
	booked, err := worker.Book(ctx, OperationLeaseBookRequest{
		OpID: opID, RegionCode: 1, Role: "management", ReplicaSetID: uuid.New(),
		Attempt: 1, FactorU: uuid.New(), Scope: scope, ReplicaNodes: []string{"c1-node"},
	})
	require.NoError(t, err)

	viewA, err := db.New(pool).GetOperationLease(ctx, domain.ToUUID(opID))
	require.NoError(t, err)
	viewB, err := db.New(pool).GetOperationLease(ctx, domain.ToUUID(opID))
	require.NoError(t, err)
	require.Equal(t, viewA.DeadlineAt.Time, viewB.DeadlineAt.Time)
	require.Equal(t, booked.Lease.DeadlineAt.Time, viewA.DeadlineAt.Time)
}

func testConditionC2FencingEpoch(t *testing.T) {
	dir := t.TempDir()
	store, err := NewLeaseFencingStore(dir)
	require.NoError(t, err)

	epoch, err := store.Next()
	require.NoError(t, err)
	require.Equal(t, int64(1), epoch)

	reloaded, err := NewLeaseFencingStore(dir)
	require.NoError(t, err)
	require.Equal(t, uint64(1), reloaded.Floor())

	require.NoError(t, store.Validate(1))
	require.ErrorIs(t, store.Validate(0), ErrStaleFencingEpoch)

	_, err = store.Next()
	require.NoError(t, err)
	require.ErrorIs(t, store.Validate(1), ErrStaleFencingEpoch)
}

func testConditionC3QuorumBook(t *testing.T) {
	ctx := context.Background()
	pool, cleanup := database.SetupTestDB(t)
	t.Cleanup(cleanup)

	cfg := &config.Config{MultiRegionEnabled: true, RegionCode: 1, NodeID: "c3-coord", OpLeaseTimeoutSec: 30}
	svc := newBareService(t, pool, nil, cfg)
	worker := NewOperationLeaseWorker(svc)
	opID := uuid.New()
	scope := dedup.NewAdapter(pool, 1, 1).RegionScope(dedupkey.RelaySourceID(1), 8, 8)
	base := OperationLeaseBookRequest{
		OpID: opID, RegionCode: 1, Role: "management", ReplicaSetID: uuid.New(),
		Attempt: 1, FactorU: uuid.New(), Scope: scope,
		ReplicaNodes: []string{"c3-a", "c3-b", "c3-c"},
		BookAckNodes: []string{},
	}

	_, err := worker.Book(ctx, base)
	require.ErrorIs(t, err, ErrLeaseQuorumNotMet)

	status, err := worker.AckBook(ctx, opID, "c3-a")
	require.NoError(t, err)
	require.False(t, status.QuorumMet)

	status, err = worker.AckBook(ctx, opID, "c3-b")
	require.NoError(t, err)
	require.True(t, status.QuorumMet)
	require.Equal(t, int32(2), status.AckCount)
}

func testConditionC4DeadlineFromPG(t *testing.T) {
	ctx := context.Background()
	pool, cleanup := database.SetupTestDB(t)
	t.Cleanup(cleanup)

	cfg := &config.Config{MultiRegionEnabled: true, RegionCode: 1, NodeID: "c4-node", OpLeaseTimeoutSec: 25}
	svc := newBareService(t, pool, nil, cfg)
	worker := NewOperationLeaseWorker(svc)

	before := time.Now().UTC()
	opID := uuid.New()
	scope := dedup.NewAdapter(pool, 1, 1).RegionScope(dedupkey.RelaySourceID(1), 9, 9)
	booked, err := worker.Book(ctx, OperationLeaseBookRequest{
		OpID: opID, RegionCode: 1, Role: "management", ReplicaSetID: uuid.New(),
		Attempt: 1, FactorU: uuid.New(), Scope: scope, ReplicaNodes: []string{"c4-node"},
	})
	require.NoError(t, err)
	after := time.Now().UTC()

	deadline := booked.Lease.DeadlineAt.Time
	require.True(t, deadline.After(before.Add(24*time.Second)))
	require.True(t, deadline.Before(after.Add(26*time.Second)))
}

func testConditionC5AttemptInScope(t *testing.T) {
	base := dedupkey.Scope{
		RegionID:    dedupkey.RegionUUID(1),
		SourceID:    dedupkey.RelaySourceID(1),
		SourceEpoch: 1,
		SeqStart:    100,
		SeqEnd:      100,
	}
	scope1 := ScopeWithAttempt(base, 1)
	scope2 := ScopeWithAttempt(base, 2)
	require.Equal(t, int64(100), scope1.SeqStart)
	require.Equal(t, int64(100002), scope2.SeqStart)

	factorU := uuid.New()
	key1 := dedupkey.FormatCanonical(scope1, factorU, uuid.Nil)
	key2 := dedupkey.FormatCanonical(scope2, factorU, uuid.Nil)
	require.NotEqual(t, key1, key2)
}

func testConditionC6RenewMax(t *testing.T) {
	ctx := context.Background()
	pool, cleanup := database.SetupTestDB(t)
	t.Cleanup(cleanup)

	cfg := &config.Config{
		MultiRegionEnabled: true,
		RegionCode:         1,
		NodeID:             "c6-exec",
		OpLeaseTimeoutSec:  30,
		OpLeaseMaxRenewals: 3,
	}
	svc := newBareService(t, pool, nil, cfg)
	worker := NewOperationLeaseWorker(svc)

	opID := uuid.New()
	scope := dedup.NewAdapter(pool, 1, 1).RegionScope(dedupkey.RelaySourceID(1), 12, 12)
	_, err := worker.Book(ctx, OperationLeaseBookRequest{
		OpID: opID, RegionCode: 1, Role: "management", ReplicaSetID: uuid.New(),
		Attempt: 1, FactorU: uuid.New(), Scope: scope, ReplicaNodes: []string{"c6-exec"},
	})
	require.NoError(t, err)
	require.NoError(t, worker.ExecuteOp(ctx, opID, func(ctx context.Context, _ db.OperationLease, _ dedup.ClaimResult) error {
		return nil
	}))

	_, err = pool.Exec(ctx, `
		UPDATE operation_leases
		SET lease_state = 'executing', executor_node_id = 'c6-exec', renew_count = 0,
		    deadline_at = NOW() + INTERVAL '30 seconds'
		WHERE op_id = $1`, domain.ToUUID(opID))
	require.NoError(t, err)

	for i := 0; i < 3; i++ {
		renewed, err := worker.RenewLease(ctx, opID)
		require.NoError(t, err)
		require.Equal(t, int32(i+1), renewed.RenewCount)
	}
	_, err = worker.RenewLease(ctx, opID)
	require.Error(t, err)
}

func testConditionC7OpKeyShed(t *testing.T) {
	ctx := context.Background()
	pool, cleanup := database.SetupTestDB(t)
	t.Cleanup(cleanup)

	cfg := &config.Config{MultiRegionEnabled: true, RegionCode: 1, NodeID: "c7-node", OpLeaseTimeoutSec: 30}
	svc := newBareService(t, pool, nil, cfg)
	worker := NewOperationLeaseWorker(svc)
	worker.SetOpKeyPoolGate(stubOpKeyGate{shed: true})

	opID := uuid.New()
	scope := dedup.NewAdapter(pool, 1, 1).RegionScope(dedupkey.RelaySourceID(1), 13, 13)
	_, err := worker.Book(ctx, OperationLeaseBookRequest{
		OpID: opID, RegionCode: 1, Role: "management", ReplicaSetID: uuid.New(),
		Attempt: 1, FactorU: uuid.New(), Scope: scope, ReplicaNodes: []string{"c7-node"},
	})
	require.ErrorIs(t, err, ErrOpKeyPoolShed)
}

func testConditionC8JanitorLeader(t *testing.T) {
	ctx := context.Background()
	pool, cleanup := database.SetupTestDB(t)
	t.Cleanup(cleanup)

	cfg := &config.Config{MultiRegionEnabled: true, RegionCode: 1, NodeID: "c8-a", OpLeaseTimeoutSec: 1}
	svc := newBareService(t, pool, nil, cfg)
	workerA := NewOperationLeaseWorker(svc)
	workerB := NewOperationLeaseWorker(svc)
	workerB.nodeID = "c8-b"

	opID := uuid.New()
	scope := dedup.NewAdapter(pool, 1, 1).RegionScope(dedupkey.RelaySourceID(1), 14, 14)
	_, err := workerA.Book(ctx, OperationLeaseBookRequest{
		OpID: opID, RegionCode: 1, Role: "management", ReplicaSetID: uuid.New(),
		Attempt: 1, FactorU: uuid.New(), Scope: scope, ReplicaNodes: []string{"c8-a"},
	})
	require.NoError(t, err)

	time.Sleep(1100 * time.Millisecond)

	var wg sync.WaitGroup
	var expiredA, expiredB int32
	var errA, errB error
	wg.Add(2)
	go func() {
		defer wg.Done()
		expiredA, errA = workerA.RunJanitor(ctx)
	}()
	go func() {
		defer wg.Done()
		expiredB, errB = workerB.RunJanitor(ctx)
	}()
	wg.Wait()
	require.NoError(t, errA)
	require.NoError(t, errB)
	require.Equal(t, int32(1), expiredA+expiredB)

	lease, err := db.New(pool).GetOperationLease(ctx, domain.ToUUID(opID))
	require.NoError(t, err)
	require.Equal(t, string(LeaseStateExpired), lease.LeaseState)
}
