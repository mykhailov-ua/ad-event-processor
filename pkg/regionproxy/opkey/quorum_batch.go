package opkey

import (
	"context"

	"espx/pkg/regionproxy/quorum"

	"github.com/redis/go-redis/v9"
)

// BatchCommitter gates opkey batch forward on Redis quorum ACKs when configured.
type BatchCommitter struct {
	rdb       redis.UniversalClient
	nodeID    string
	replicas  []string
	committed uint64
}

// NewBatchCommitter builds a quorum gate for one proxy node.
func NewBatchCommitter(rdb redis.UniversalClient, nodeID string, replicas []string) *BatchCommitter {
	return &BatchCommitter{
		rdb:      rdb,
		nodeID:   nodeID,
		replicas: replicas,
	}
}

// Committed returns batches that passed quorum and executing claim.
func (c *BatchCommitter) Committed() uint64 {
	if c == nil {
		return 0
	}
	return c.committed
}

// PrepareForward transitions derived -> booked -> executing only after 2-of-3 quorum ACKs.
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
	replicaCount := len(c.replicas)
	if replicaCount == 0 {
		replicaCount = 1
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

// Complete marks executing -> completed in Redis after a successful uplink ACK.
func (c *BatchCommitter) Complete(ctx context.Context, slot *Slot) {
	if c == nil || c.rdb == nil || slot == nil {
		return
	}
	if err := quorum.Transition(ctx, c.rdb, slot.OpID, quorum.StateExecuting, quorum.StateCompleted); err == nil {
		c.committed++
	}
}
