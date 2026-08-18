package controlplane

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/internal/database"
	"github.com/bidshard/ad-event-processor/internal/domain"
	db "github.com/bidshard/ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func TestOperationLeaseClaimExecuting_SingleWinnerUnderContention(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	pool, cleanup := database.SetupTestDB(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	q := db.New(pool)

	opID := uuid.New()
	replicaSetID := uuid.New()
	factorU := uuid.New()

	_, err := q.InsertOperationLease(ctx, db.InsertOperationLeaseParams{
		OpID:         domain.ToUUID(opID),
		RegionCode:   1,
		Role:         "tracker",
		ReplicaSetID: domain.ToUUID(replicaSetID),
		Attempt:      1,
		FactorU:      domain.ToUUID(factorU),
		DedupScope:   []byte(`{}`),
		DeadlineAt:   pgtype.Timestamptz{Time: time.Now().Add(30 * time.Second), Valid: true},
	})
	require.NoError(t, err)

	const contenders = 32
	var wg sync.WaitGroup
	winners := make(chan string, contenders)
	var claimErrs atomic.Int32

	for i := range contenders {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			nodeID := fmt.Sprintf("node-%02d", i)
			rows, err := q.OperationLeaseClaimExecuting(ctx, db.OperationLeaseClaimExecutingParams{
				OpID:         domain.ToUUID(opID),
				NodeID:       nodeID,
				FencingEpoch: int64(i + 1),
			})
			if err != nil {
				claimErrs.Add(1)
				return
			}
			if len(rows) > 0 {
				winners <- nodeID
			}
		}(i)
	}
	wg.Wait()
	close(winners)

	require.Zero(t, claimErrs.Load(), "claim must not error under contention")

	var winnerNodes []string
	for nodeID := range winners {
		winnerNodes = append(winnerNodes, nodeID)
	}
	require.Len(t, winnerNodes, 1, "exactly one CAS winner")

	lease, err := q.GetOperationLease(ctx, domain.ToUUID(opID))
	require.NoError(t, err)
	require.Equal(t, string(LeaseStateExecuting), lease.LeaseState)
	require.True(t, lease.ExecutorNodeID.Valid)
	require.Equal(t, winnerNodes[0], lease.ExecutorNodeID.String)
}

func TestOperationLease_CompleteAndExpire(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	pool, cleanup := database.SetupTestDB(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	q := db.New(pool)

	opID := uuid.New()
	replicaSetID := uuid.New()

	_, err := q.InsertOperationLease(ctx, db.InsertOperationLeaseParams{
		OpID:         domain.ToUUID(opID),
		RegionCode:   1,
		Role:         "tracker",
		ReplicaSetID: domain.ToUUID(replicaSetID),
		Attempt:      1,
		FactorU:      domain.ToUUID(uuid.New()),
		DedupScope:   []byte(`{"scope":"test"}`),
		DeadlineAt:   pgtype.Timestamptz{Time: time.Now().Add(30 * time.Second), Valid: true},
	})
	require.NoError(t, err)

	rows, err := q.OperationLeaseClaimExecuting(ctx, db.OperationLeaseClaimExecutingParams{
		OpID:         domain.ToUUID(opID),
		NodeID:       "executor-1",
		FencingEpoch: 7,
	})
	require.NoError(t, err)
	require.Len(t, rows, 1)

	completed, err := q.CompleteOperationLease(ctx, domain.ToUUID(opID))
	require.NoError(t, err)
	require.Equal(t, string(LeaseStateCompleted), completed.LeaseState)
	require.True(t, completed.CompletedAt.Valid)

	staleID := uuid.New()
	_, err = q.InsertOperationLease(ctx, db.InsertOperationLeaseParams{
		OpID:         domain.ToUUID(staleID),
		RegionCode:   1,
		Role:         "tracker",
		ReplicaSetID: domain.ToUUID(uuid.New()),
		Attempt:      1,
		FactorU:      domain.ToUUID(uuid.New()),
		DedupScope:   []byte(`{}`),
		DeadlineAt:   pgtype.Timestamptz{Time: time.Now().Add(-time.Second), Valid: true},
	})
	require.NoError(t, err)

	expiredCount, err := q.OperationLeaseExpireStale(ctx, 100)
	require.NoError(t, err)
	require.Equal(t, int32(1), expiredCount)

	stale, err := q.GetOperationLease(ctx, domain.ToUUID(staleID))
	require.NoError(t, err)
	require.Equal(t, string(LeaseStateExpired), stale.LeaseState)
}

func TestValidLeaseState(t *testing.T) {
	require.True(t, ValidLeaseState("booked"))
	require.True(t, ValidLeaseState("executing"))
	require.False(t, ValidLeaseState("BOOKED"))
	require.False(t, ValidLeaseState(""))
}
