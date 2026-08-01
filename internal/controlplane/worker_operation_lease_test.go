package controlplane

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"espx/internal/config"
	"espx/internal/database"
	"espx/internal/dedup"
	"espx/internal/domain"
	db "espx/internal/domain/db"
	"espx/pkg/dedupkey"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestOperationLeaseWorker_BookedExecutingCompleted(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	ctx := context.Background()
	pool, cleanup := database.SetupTestDB(t)
	t.Cleanup(cleanup)

	cfg := &config.Config{
		MultiRegionEnabled: true,
		RegionCode:         1,
		NodeID:             "mgmt-lease-1",
		OpLeaseTimeoutSec:  30,
	}
	svc := newBareService(t, pool, nil, cfg)
	worker := NewOperationLeaseWorker(svc)

	opID := uuid.New()
	replicaSetID := uuid.New()
	factorU := uuid.New()
	scope := dedup.NewAdapter(pool, 1, 1).RegionScope(dedupkey.RelaySourceID(1), 42, 42)

	_, err := worker.Book(ctx, OperationLeaseBookRequest{
		OpID:         opID,
		RegionCode:   1,
		Role:         "management",
		ReplicaSetID: replicaSetID,
		Attempt:      1,
		FactorU:      factorU,
		Scope:        scope,
		ReplicaNodes: []string{cfg.NodeID},
	})
	require.NoError(t, err)

	var applied atomic.Bool
	err = worker.ExecuteOp(ctx, opID, func(ctx context.Context, lease db.OperationLease, claim dedup.ClaimResult) error {
		require.Equal(t, string(LeaseStateExecuting), lease.LeaseState)
		require.Equal(t, dedup.OutcomeConfirmed, claim.Outcome)
		applied.Store(true)
		return nil
	})
	require.NoError(t, err)
	require.True(t, applied.Load())

	lease, err := db.New(pool).GetOperationLease(ctx, domain.ToUUID(opID))
	require.NoError(t, err)
	require.Equal(t, string(LeaseStateCompleted), lease.LeaseState)
	require.True(t, lease.CompletedAt.Valid)
}

func TestOperationLeaseWorker_DualCASOneExecutor(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	ctx := context.Background()
	pool, cleanup := database.SetupTestDB(t)
	t.Cleanup(cleanup)

	cfg := &config.Config{
		MultiRegionEnabled: true,
		RegionCode:         1,
		NodeID:             "mgmt-cas-coordinator",
		OpLeaseTimeoutSec:  30,
	}
	svc := newBareService(t, pool, nil, cfg)
	worker := NewOperationLeaseWorker(svc)

	opID := uuid.New()
	scope := dedup.NewAdapter(pool, 1, 1).RegionScope(dedupkey.RelaySourceID(1), 7, 7)
	_, err := worker.Book(ctx, OperationLeaseBookRequest{
		OpID:         opID,
		RegionCode:   1,
		Role:         "management",
		ReplicaSetID: uuid.New(),
		Attempt:      1,
		FactorU:      uuid.New(),
		Scope:        scope,
		ReplicaNodes: []string{"node-a", "node-b", "node-c"},
		BookAckNodes: []string{"node-a", "node-b"},
	})
	require.NoError(t, err)

	const contenders = 32
	var wg sync.WaitGroup
	for i := 0; i < contenders; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			nodeWorker := NewOperationLeaseWorker(svc)
			nodeWorker.nodeID = fmt.Sprintf("node-%c", 'a'+i%3)
			_ = nodeWorker.ExecuteOp(ctx, opID, func(ctx context.Context, _ db.OperationLease, _ dedup.ClaimResult) error {
				return nil
			})
		}(i)
	}
	wg.Wait()

	lease, err := db.New(pool).GetOperationLease(ctx, domain.ToUUID(opID))
	require.NoError(t, err)
	require.Equal(t, string(LeaseStateCompleted), lease.LeaseState)
	require.True(t, lease.ExecutorNodeID.Valid)
}
