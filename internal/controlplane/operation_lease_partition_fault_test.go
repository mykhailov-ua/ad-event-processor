package controlplane

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/pkg/faultproof"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/database"
	"github.com/bidshard/ad-event-processor/internal/dedup"
	"github.com/bidshard/ad-event-processor/internal/domain"
	db "github.com/bidshard/ad-event-processor/internal/domain/db"
	"github.com/bidshard/ad-event-processor/pkg/dedupkey"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

const leaseFaultContainerStopTimeout = 10 * time.Second

func TestFault_OperationLease_PGPartition(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	ctx := context.Background()
	pool, cleanup := database.SetupTestDB(t)
	t.Cleanup(cleanup)

	cfg := &config.Config{
		MultiRegionEnabled: true,
		RegionCode:         1,
		NodeID:             "mgmt-partition",
		OpLeaseTimeoutSec:  1,
	}
	svc := newBareService(t, pool, nil, cfg)
	worker := NewOperationLeaseWorker(svc)

	opID := uuid.New()
	scope := dedup.NewAdapter(pool, 1, 1).RegionScope(dedupkey.RelaySourceID(1), 99, 99)
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

	time.Sleep(1100 * time.Millisecond)
	expired, err := worker.RunJanitor(ctx)
	require.NoError(t, err)
	require.GreaterOrEqual(t, expired, int32(1))

	lease, err := db.New(pool).GetOperationLease(ctx, domain.ToUUID(opID))
	require.NoError(t, err)
	require.Equal(t, string(LeaseStateExpired), lease.LeaseState)

	faultproof.Log(t, "mr_lease_pg_partition", map[string]string{
		"subsystem":   "operation_lease",
		"op_id":       opID.String(),
		"lease_state": lease.LeaseState,
		"budget_ok":   "true",
		"baseline_ok": "true",
	})
}

func TestFault_OperationLease_PGStopDuringExecuting(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	ctx := context.Background()
	infra, cleanup := database.SetupTestDBInfra(t)
	t.Cleanup(cleanup)

	cfg := &config.Config{
		MultiRegionEnabled: true,
		RegionCode:         1,
		NodeID:             "mgmt-exec-partition",
		OpLeaseTimeoutSec:  1,
	}
	svc := newBareService(t, infra.Pool, nil, cfg)
	worker := NewOperationLeaseWorker(svc)
	worker.SetRenewHeartbeat(false)

	opID := uuid.New()
	scope := dedup.NewAdapter(infra.Pool, 1, 1).RegionScope(dedupkey.RelaySourceID(1), 100, 100)
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

	started := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- worker.ExecuteOp(ctx, opID, func(ctx context.Context, _ db.OperationLease, _ dedup.ClaimResult) error {
			close(started)
			time.Sleep(3 * time.Second)
			return nil
		})
	}()
	<-started

	stopLeasePGContainer(t, infra)
	require.Eventually(t, func() bool {
		return infra.Pool.Ping(ctx) != nil
	}, 15*time.Second, 200*time.Millisecond, "postgres must be unreachable during partition")

	time.Sleep(1200 * time.Millisecond)

	startLeasePGContainer(t, infra)
	refreshLeasePGPool(t, infra)
	svc.SetPool(infra.Pool)

	expired, err := worker.RunJanitor(ctx)
	require.NoError(t, err)
	require.GreaterOrEqual(t, expired, int32(1))

	lease, err := db.New(infra.Pool).GetOperationLease(ctx, domain.ToUUID(opID))
	require.NoError(t, err)
	require.Equal(t, string(LeaseStateExpired), lease.LeaseState)

	select {
	case execErr := <-done:
		require.Error(t, execErr, "executor must not complete after lease expiry")
	case <-time.After(5 * time.Second):
		t.Fatal("executor did not return after partition heal")
	}

	faultproof.Log(t, "mr_lease_pg_partition_executing", map[string]string{
		"subsystem":    "operation_lease",
		"op_id":        opID.String(),
		"lease_state":  lease.LeaseState,
		"budget_ok":    "true",
		"baseline_ok":  "true",
		"fault_verify": "postgres_stop_during_executing",
	})
}

func TestFault_OperationLease_GhostExecutor(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	ctx := context.Background()
	pool, cleanup := database.SetupTestDB(t)
	t.Cleanup(cleanup)
	rdb, cleanupRedis := database.SetupTestRedis(t)
	t.Cleanup(cleanupRedis)

	cfg := &config.Config{
		MultiRegionEnabled: true,
		RegionCode:         1,
		NodeID:             "ghost-primary",
		OpLeaseTimeoutSec:  30,
	}
	svc := newBareService(t, pool, []redis.UniversalClient{rdb}, cfg)
	worker := NewOperationLeaseWorker(svc)

	opID := uuid.New()
	factorU := uuid.New()
	scope := dedup.NewAdapter(pool, 1, 1).RegionScope(dedupkey.RelaySourceID(1), 11, 11)
	_, err := worker.Book(ctx, OperationLeaseBookRequest{
		OpID:         opID,
		RegionCode:   1,
		Role:         "management",
		ReplicaSetID: uuid.New(),
		Attempt:      1,
		FactorU:      factorU,
		Scope:        scope,
		ReplicaNodes: []string{cfg.NodeID, "ghost-standby"},
		BookAckNodes: []string{cfg.NodeID, "ghost-standby"},
	})
	require.NoError(t, err)

	redisKey := "lease:ghost:" + opID.String()
	err = worker.ExecuteOp(ctx, opID, func(ctx context.Context, _ db.OperationLease, claim dedup.ClaimResult) error {
		require.Equal(t, dedup.OutcomeConfirmed, claim.Outcome)
		ok, err := rdb.SetNX(ctx, redisKey, "1", time.Hour).Result()
		require.NoError(t, err)
		require.True(t, ok)
		return nil
	})
	require.NoError(t, err)

	ghost := NewOperationLeaseWorker(svc)
	ghost.nodeID = "ghost-standby"
	err = ghost.ExecuteOp(ctx, opID, func(ctx context.Context, _ db.OperationLease, claim dedup.ClaimResult) error {
		t.Fatalf("ghost executor must not apply side effects on completed lease, outcome=%s", claim.Outcome)
		return nil
	})
	require.NoError(t, err)

	val, err := rdb.Get(ctx, redisKey).Result()
	require.NoError(t, err)
	require.Equal(t, "1", val)

	var proposalCount int
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM dedup_key_proposals`).Scan(&proposalCount))
	require.Equal(t, 1, proposalCount)

	faultproof.Log(t, "mr_lease_ghost_executor", map[string]string{
		"subsystem":     "operation_lease",
		"op_id":         opID.String(),
		"redis_budget":  val,
		"proposal_rows": strconv.Itoa(proposalCount),
		"baseline_ok":   "true",
	})
}

func TestFault_OperationLease_DualCAS(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	ctx := context.Background()
	pool, cleanup := database.SetupTestDB(t)
	t.Cleanup(cleanup)

	cfg := &config.Config{
		MultiRegionEnabled: true,
		RegionCode:         1,
		NodeID:             "cas-primary",
		OpLeaseTimeoutSec:  30,
	}
	svc := newBareService(t, pool, nil, cfg)
	coordinator := NewOperationLeaseWorker(svc)

	opID := uuid.New()
	scope := dedup.NewAdapter(pool, 1, 1).RegionScope(dedupkey.RelaySourceID(1), 21, 21)
	_, err := coordinator.Book(ctx, OperationLeaseBookRequest{
		OpID:         opID,
		RegionCode:   1,
		Role:         "management",
		ReplicaSetID: uuid.New(),
		Attempt:      1,
		FactorU:      uuid.New(),
		Scope:        scope,
		ReplicaNodes: []string{"cas-a", "cas-b", "cas-c"},
		BookAckNodes: []string{"cas-a", "cas-b"},
	})
	require.NoError(t, err)

	const contenders = 32
	var wg sync.WaitGroup
	executionCount := make(chan struct{}, contenders)
	for i := range contenders {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			nodeWorker := NewOperationLeaseWorker(svc)
			nodeWorker.nodeID = fmt.Sprintf("cas-%c", 'a'+i%3)
			_ = nodeWorker.ExecuteOp(ctx, opID, func(ctx context.Context, lease db.OperationLease, _ dedup.ClaimResult) error {
				if lease.LeaseState == string(LeaseStateExecuting) {
					executionCount <- struct{}{}
				}
				return nil
			})
		}(i)
	}
	wg.Wait()
	close(executionCount)

	var executions int
	for range executionCount {
		executions++
	}
	require.Equal(t, 1, executions)

	lease, err := db.New(pool).GetOperationLease(ctx, domain.ToUUID(opID))
	require.NoError(t, err)
	require.Equal(t, string(LeaseStateCompleted), lease.LeaseState)
	require.True(t, lease.ExecutorNodeID.Valid)

	faultproof.Log(t, "mr_lease_dual_cas", map[string]string{
		"subsystem":       "operation_lease",
		"op_id":           opID.String(),
		"execution_count": "1",
		"executor_node":   lease.ExecutorNodeID.String,
		"baseline_ok":     "true",
	})
}

func stopLeasePGContainer(t *testing.T, infra *database.TestDBInfra) {
	t.Helper()
	timeout := leaseFaultContainerStopTimeout
	require.NoError(t, infra.PGContainer.Stop(context.Background(), &timeout))
}

func startLeasePGContainer(t *testing.T, infra *database.TestDBInfra) {
	t.Helper()
	require.NoError(t, infra.PGContainer.Start(context.Background()))
}

func refreshLeasePGPool(t *testing.T, infra *database.TestDBInfra) {
	t.Helper()
	ctx := context.Background()
	infra.Pool.Close()
	connStr, err := infra.PGContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	pool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)
	infra.Pool = pool
	require.Eventually(t, func() bool {
		return pool.Ping(ctx) == nil
	}, 30*time.Second, 200*time.Millisecond)
}
