package management

import (
	"espx/pkg/faultproof"

	"context"
	"os"
	"path/filepath"
	"testing"

	"espx/internal/config"
	"espx/internal/database"
	"espx/internal/dedup"
	db "espx/internal/ingestion/sqlc"
	"espx/pkg/dedupkey"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestLeaseFencingRegistry_PerReplicaSetIsolation(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	reg, err := NewLeaseFencingRegistry(base)
	require.NoError(t, err)

	setA := uuid.New()
	setB := uuid.New()

	epochA, err := reg.Next(setA)
	require.NoError(t, err)
	require.Equal(t, int64(1), epochA)

	epochB, err := reg.Next(setB)
	require.NoError(t, err)
	require.Equal(t, int64(1), epochB)

	_, err = reg.Next(setA)
	require.NoError(t, err)
	require.ErrorIs(t, reg.Validate(setA, 1), ErrStaleFencingEpoch)
	require.NoError(t, reg.Validate(setB, 1))

	pathA := filepath.Join(base, setA.String(), leaseFencingEpochFile)
	pathB := filepath.Join(base, setB.String(), leaseFencingEpochFile)
	_, err = os.Stat(pathA)
	require.NoError(t, err)
	_, err = os.Stat(pathB)
	require.NoError(t, err)
}

func TestOperationLease_StaleFencingEpochRejectsExecutor(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	reg, err := NewLeaseFencingRegistry(base)
	require.NoError(t, err)

	replicaSetID := uuid.New()
	staleEpoch, err := reg.Next(replicaSetID)
	require.NoError(t, err)
	require.Equal(t, int64(1), staleEpoch)

	_, err = reg.Next(replicaSetID)
	require.NoError(t, err)

	require.ErrorIs(t, reg.Validate(replicaSetID, staleEpoch), ErrStaleFencingEpoch)
}

func TestFault_OperationLease_GhostExecutorFencingProof(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test")
	}

	ctx := context.Background()
	pool, cleanup := database.SetupTestDB(t)
	t.Cleanup(cleanup)

	fencingBase := t.TempDir()
	cfg := &config.Config{
		MultiRegionEnabled: true,
		RegionCode:         1,
		NodeID:             "fence-ghost",
		OpLeaseTimeoutSec:  30,
		OpLeaseFencingDir:  fencingBase,
	}
	svc := newBareService(t, pool, nil, cfg)
	worker := NewOperationLeaseWorker(svc)

	replicaSetID := uuid.New()
	opID := uuid.New()
	scope := dedup.NewAdapter(pool, 1, 1).RegionScope(dedupkey.RelaySourceID(1), 41, 41)
	_, err := worker.Book(ctx, OperationLeaseBookRequest{
		OpID:         opID,
		RegionCode:   1,
		Role:         "management",
		ReplicaSetID: replicaSetID,
		Attempt:      1,
		FactorU:      uuid.New(),
		Scope:        scope,
		ReplicaNodes: []string{cfg.NodeID, "fence-standby"},
		BookAckNodes: []string{cfg.NodeID, "fence-standby"},
	})
	require.NoError(t, err)

	err = worker.ExecuteOp(ctx, opID, func(ctx context.Context, _ db.OperationLease, _ dedup.ClaimResult) error {
		return nil
	})
	require.NoError(t, err)

	ghost := NewOperationLeaseWorker(svc)
	ghost.nodeID = "fence-standby"
	err = ghost.ExecuteOp(ctx, opID, func(ctx context.Context, _ db.OperationLease, _ dedup.ClaimResult) error {
		t.Fatal("ghost must not apply after primary completed")
		return nil
	})
	require.NoError(t, err)

	fencingPath := filepath.Join(fencingBase, replicaSetID.String(), leaseFencingEpochFile)
	_, err = os.Stat(fencingPath)
	require.NoError(t, err)

	faultproof.Log(t, "mr_lease_ghost_executor", map[string]string{
		"subsystem":      "operation_lease",
		"op_id":          opID.String(),
		"replica_set_id": replicaSetID.String(),
		"fencing_epoch":  "1",
		"baseline_ok":    "true",
	})
}
