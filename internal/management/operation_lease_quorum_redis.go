package management

import (
	"context"
	"errors"

	"espx/pkg/regionproxy/quorum"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
)

func (w *OperationLeaseWorker) pgAvailable(ctx context.Context) bool {
	if w == nil || w.svc == nil || w.svc.pool == nil {
		return false
	}
	return w.svc.pool.Ping(ctx) == nil
}

func (w *OperationLeaseWorker) quorumRDB() redis.UniversalClient {
	if w == nil || w.svc == nil {
		return nil
	}
	return PickHealthyControlShard(w.svc.rdbs)
}

func (w *OperationLeaseWorker) bookRedis(ctx context.Context, req OperationLeaseBookRequest) (OperationLeaseBookResult, error) {
	empty := OperationLeaseBookResult{}
	rdb := w.quorumRDB()
	if rdb == nil {
		return empty, ErrLeaseQuorumNotMet
	}
	opID := opIDBytes(req.OpID)
	replicas := req.ReplicaNodes
	if len(replicas) == 0 {
		replicas = []string{w.nodeID}
	}
	for _, n := range req.BookAckNodes {
		st, err := quorum.AckBook(ctx, rdb, opID, len(replicas), n)
		if err != nil {
			return empty, err
		}
		if st.QuorumMet {
			return operationLeaseBookResultFromQuorum(st), nil
		}
	}
	st, err := quorum.Book(ctx, rdb, opID, replicas, w.nodeID)
	if err != nil {
		return empty, err
	}
	result := operationLeaseBookResultFromQuorum(st)
	if !result.QuorumMet {
		return result, ErrLeaseQuorumNotMet
	}
	return result, nil
}

func (w *OperationLeaseWorker) ackBookRedis(ctx context.Context, opID uuid.UUID, nodeID string, replicaCount int) (OperationLeaseBookResult, error) {
	empty := OperationLeaseBookResult{}
	rdb := w.quorumRDB()
	if rdb == nil {
		return empty, ErrLeaseQuorumNotMet
	}
	st, err := quorum.AckBook(ctx, rdb, opIDBytes(opID), replicaCount, nodeID)
	if err != nil {
		return empty, err
	}
	return operationLeaseBookResultFromQuorum(st), nil
}

func (w *OperationLeaseWorker) quorumStatusRedis(ctx context.Context, opID uuid.UUID, replicaCount int) (OperationLeaseBookResult, error) {
	rdb := w.quorumRDB()
	if rdb == nil {
		return OperationLeaseBookResult{}, ErrLeaseQuorumNotMet
	}
	st, err := quorum.ReadStatus(ctx, rdb, opIDBytes(opID), replicaCount)
	if err != nil {
		return OperationLeaseBookResult{}, err
	}
	return operationLeaseBookResultFromQuorum(st), nil
}

func operationLeaseBookResultFromQuorum(st quorum.Status) OperationLeaseBookResult {
	return OperationLeaseBookResult{
		AckCount:  st.AckCount,
		Quorum:    st.Quorum,
		QuorumMet: st.QuorumMet,
	}
}

func opIDBytes(id uuid.UUID) [16]byte {
	var out [16]byte
	copy(out[:], id[:])
	return out
}

func isPgUnavailable(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	return true
}
