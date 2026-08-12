package controlplane

import (
	"context"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/database"
	"github.com/bidshard/ad-event-processor/internal/dedup"
	"github.com/bidshard/ad-event-processor/internal/domain"
	db "github.com/bidshard/ad-event-processor/internal/domain/db"
	"github.com/bidshard/ad-event-processor/pkg/dedupkey"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestOperationLeaseRenew_WithoutRenewExpires(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	ctx := context.Background()
	pool, cleanup := database.SetupTestDB(t)
	t.Cleanup(cleanup)

	cfg := &config.Config{
		MultiRegionEnabled: true,
		RegionCode:         1,
		NodeID:             "renew-expire",
		OpLeaseTimeoutSec:  1,
	}
	svc := newBareService(t, pool, nil, cfg)
	worker := NewOperationLeaseWorker(svc)

	opID, err := bookAndClaimLease(ctx, t, worker, 1)
	require.NoError(t, err)

	time.Sleep(1100 * time.Millisecond)
	expired, err := worker.RunJanitor(ctx)
	require.NoError(t, err)
	require.GreaterOrEqual(t, expired, int32(1))

	lease, err := db.New(pool).GetOperationLease(ctx, domain.ToUUID(opID))
	require.NoError(t, err)
	require.Equal(t, string(LeaseStateExpired), lease.LeaseState)
}

func TestOperationLeaseRenew_ManualRenewPreventsExpiry(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	ctx := context.Background()
	pool, cleanup := database.SetupTestDB(t)
	t.Cleanup(cleanup)

	cfg := &config.Config{
		MultiRegionEnabled: true,
		RegionCode:         1,
		NodeID:             "renew-extend",
		OpLeaseTimeoutSec:  1,
		OpLeaseMaxRenewals: 3,
	}
	svc := newBareService(t, pool, nil, cfg)
	worker := NewOperationLeaseWorker(svc)

	opID, err := bookAndClaimLease(ctx, t, worker, 1)
	require.NoError(t, err)

	time.Sleep(800 * time.Millisecond)
	renewed, err := worker.RenewLease(ctx, opID)
	require.NoError(t, err)
	require.Equal(t, int32(1), renewed.RenewCount)
	require.True(t, renewed.DeadlineAt.Time.After(time.Now().UTC()))

	var hookOpID uuid.UUID
	worker.SetLeaseRenewHook(func(id uuid.UUID) {
		hookOpID = id
	})
	_, err = worker.RenewLease(ctx, opID)
	require.NoError(t, err)
	require.Equal(t, opID, hookOpID)

	expired, err := worker.RunJanitor(ctx)
	require.NoError(t, err)
	require.Equal(t, int32(0), expired)

	lease, err := db.New(pool).GetOperationLease(ctx, domain.ToUUID(opID))
	require.NoError(t, err)
	require.Equal(t, string(LeaseStateExecuting), lease.LeaseState)
}

func TestOperationLeaseRenew_HeartbeatSurvivesSlowExecute(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	ctx := context.Background()
	pool, cleanup := database.SetupTestDB(t)
	t.Cleanup(cleanup)

	cfg := &config.Config{
		MultiRegionEnabled: true,
		RegionCode:         1,
		NodeID:             "renew-slow",
		OpLeaseTimeoutSec:  3,
		OpLeaseMaxRenewals: 3,
	}
	svc := newBareService(t, pool, nil, cfg)
	worker := NewOperationLeaseWorker(svc)

	opID := uuid.New()
	scope := dedup.NewAdapter(pool, 1, 1).RegionScope(dedupkey.RelaySourceID(1), 50, 50)
	_, err := worker.Book(ctx, OperationLeaseBookRequest{
		OpID:         opID,
		RegionCode:   1,
		Role:         "management",
		ReplicaSetID: uuid.New(),
		Attempt:      1,
		FactorU:      uuid.New(),
		Scope:        scope,
		ReplicaNodes: []string{cfg.NodeID},
	})
	require.NoError(t, err)

	err = worker.ExecuteOp(ctx, opID, func(ctx context.Context, _ db.OperationLease, _ dedup.ClaimResult) error {
		time.Sleep(5 * time.Second)
		return nil
	})
	require.NoError(t, err)

	lease, err := db.New(pool).GetOperationLease(ctx, domain.ToUUID(opID))
	require.NoError(t, err)
	require.Equal(t, string(LeaseStateCompleted), lease.LeaseState)
	require.GreaterOrEqual(t, lease.RenewCount, int32(1))
}

func bookAndClaimLease(ctx context.Context, t *testing.T, worker *OperationLeaseWorker, seq int64) (uuid.UUID, error) {
	t.Helper()
	opID := uuid.New()
	scope := dedup.NewAdapter(worker.svc.pool, 1, 1).RegionScope(dedupkey.RelaySourceID(1), seq, seq)
	_, err := worker.Book(ctx, OperationLeaseBookRequest{
		OpID:         opID,
		RegionCode:   1,
		Role:         "management",
		ReplicaSetID: uuid.New(),
		Attempt:      1,
		FactorU:      uuid.New(),
		Scope:        scope,
		ReplicaNodes: []string{worker.nodeID},
	})
	if err != nil {
		return uuid.Nil, err
	}
	rows, err := db.New(worker.svc.pool).OperationLeaseClaimExecuting(ctx, db.OperationLeaseClaimExecutingParams{
		OpID:         domain.ToUUID(opID),
		NodeID:       worker.nodeID,
		FencingEpoch: 1,
	})
	if err != nil {
		return uuid.Nil, err
	}
	if len(rows) == 0 {
		t.Fatal("expected claim executing winner")
	}
	return opID, nil
}
