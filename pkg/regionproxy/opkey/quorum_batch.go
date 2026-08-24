package opkey

import (
	"context"

	"ad-event-processor/pkg/regionproxy/quorum"

	"github.com/redis/go-redis/v9"
)

type BatchCommitter struct {
	rdb       redis.UniversalClient
	nodeID    string
	replicas  []string
	committed uint64
}

func NewBatchCommitter(rdb redis.UniversalClient, nodeID string, replicas []string) *BatchCommitter {
	return &BatchCommitter{
		rdb:      rdb,
		nodeID:   nodeID,
		replicas: replicas,
	}
}

func (c *BatchCommitter) Committed() uint64 {
	if c == nil {
		return 0
	}
	return c.committed
}

func (c *BatchCommitter) PrepareForward(ctx context.Context, slot *Slot) (bool, error) {
	if slot == nil {
		return false, nil
	}
	if c == nil || c.rdb == nil {
		if !slot.TryBook() {
			return slot.Has(OpKeyFlagReplicaBooked), nil
		}
		return slot.TryClaimExecuting(), nil
	}
	st, err := quorum.Book(ctx, c.rdb, slot.OpID, c.replicas, c.nodeID)
	if err != nil {
		return false, err
	}
	if !st.QuorumMet {
		return false, nil
	}
	if !slot.TryBook() && !slot.Has(OpKeyFlagReplicaBooked) {
		return false, nil
	}
	if err := quorum.Transition(ctx, c.rdb, slot.OpID, quorum.StateBooked, quorum.StateExecuting); err != nil {
		return false, err
	}
	if !slot.TryClaimExecuting() {
		return false, nil
	}
	return true, nil
}

func (c *BatchCommitter) Complete(ctx context.Context, slot *Slot) {
	if c == nil || c.rdb == nil || slot == nil {
		return
	}
	if err := quorum.Transition(ctx, c.rdb, slot.OpID, quorum.StateExecuting, quorum.StateCompleted); err == nil {
		c.committed++
	}
}
