package quorum

import (
	"context"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	StateBooked    = "booked"
	StateExecuting = "executing"
	StateCompleted = "completed"

	defaultQuorum     = 2
	leaseKeyPrefix    = "espx:op:lease:"
	leaseAckKeyPrefix = "espx:op:lease:ack:"
	leaseTTL          = 48 * time.Hour
)

type Status struct {
	AckCount  int32
	Quorum    int32
	QuorumMet bool
	State     string
}

func Required(replicaCount int) int32 {
	if replicaCount <= 1 {
		return 1
	}
	return defaultQuorum
}

func Book(ctx context.Context, rdb redis.UniversalClient, opID [16]byte, replicaNodes []string, nodeID string) (Status, error) {
	if rdb == nil {
		return Status{}, fmt.Errorf("quorum book: redis unavailable")
	}
	if nodeID == "" {
		return Status{}, fmt.Errorf("quorum book: node_id required")
	}
	replicaCount := len(replicaNodes)
	if replicaCount == 0 {
		replicaCount = 1
	}
	key := leaseKey(opID)
	ackKey := leaseAckKey(opID)
	pipe := rdb.TxPipeline()
	pipe.HSetNX(ctx, key, "state", StateBooked)
	pipe.HSet(ctx, key, "replica_count", replicaCount)
	pipe.SAdd(ctx, ackKey, nodeID)
	pipe.Expire(ctx, key, leaseTTL)
	pipe.Expire(ctx, ackKey, leaseTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		return Status{}, fmt.Errorf("quorum book: %w", err)
	}
	return readStatus(ctx, rdb, opID, replicaCount)
}

func AckBook(ctx context.Context, rdb redis.UniversalClient, opID [16]byte, replicaCount int, nodeID string) (Status, error) {
	if rdb == nil {
		return Status{}, fmt.Errorf("quorum ack book: redis unavailable")
	}
	if nodeID == "" {
		return Status{}, fmt.Errorf("quorum ack book: node_id required")
	}
	if replicaCount <= 0 {
		replicaCount = 1
	}
	key := leaseKey(opID)
	ackKey := leaseAckKey(opID)
	pipe := rdb.TxPipeline()
	pipe.HSetNX(ctx, key, "state", StateBooked)
	pipe.HSet(ctx, key, "replica_count", replicaCount)
	pipe.SAdd(ctx, ackKey, nodeID)
	pipe.Expire(ctx, key, leaseTTL)
	pipe.Expire(ctx, ackKey, leaseTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		return Status{}, fmt.Errorf("quorum ack book: %w", err)
	}
	return readStatus(ctx, rdb, opID, replicaCount)
}

func Transition(ctx context.Context, rdb redis.UniversalClient, opID [16]byte, from, to string) error {
	if rdb == nil {
		return fmt.Errorf("quorum transition: redis unavailable")
	}
	key := leaseKey(opID)
	script := redis.NewScript(`
if redis.call("HGET", KEYS[1], "state") == ARGV[1] then
  redis.call("HSET", KEYS[1], "state", ARGV[2])
  return 1
end
return 0`)
	res, err := script.Run(ctx, rdb, []string{key}, from, to).Int()
	if err != nil {
		return fmt.Errorf("quorum transition: %w", err)
	}
	if res == 0 {
		return fmt.Errorf("quorum transition: state mismatch want %s", from)
	}
	return nil
}

func ReadStatus(ctx context.Context, rdb redis.UniversalClient, opID [16]byte, replicaCount int) (Status, error) {
	if replicaCount <= 0 {
		replicaCount = 1
	}
	return readStatus(ctx, rdb, opID, replicaCount)
}

func readStatus(ctx context.Context, rdb redis.UniversalClient, opID [16]byte, replicaCount int) (Status, error) {
	key := leaseKey(opID)
	ackKey := leaseAckKey(opID)
	pipe := rdb.TxPipeline()
	stateCmd := pipe.HGet(ctx, key, "state")
	countCmd := pipe.HGet(ctx, key, "replica_count")
	ackCmd := pipe.SCard(ctx, ackKey)
	if _, err := pipe.Exec(ctx); err != nil && err != redis.Nil {
		return Status{}, err
	}
	state, _ := stateCmd.Result()
	if state == "" {
		state = StateBooked
	}
	if stored, err := countCmd.Int(); err == nil && stored > 0 {
		replicaCount = stored
	}
	acks, err := ackCmd.Result()
	if err != nil {
		return Status{}, err
	}
	quorum := Required(replicaCount)
	ack32 := int32(acks)
	return Status{
		AckCount:  ack32,
		Quorum:    quorum,
		QuorumMet: ack32 >= quorum,
		State:     state,
	}, nil
}

func leaseKey(opID [16]byte) string {
	return leaseKeyPrefix + hex.EncodeToString(opID[:])
}

func leaseAckKey(opID [16]byte) string {
	return leaseAckKeyPrefix + hex.EncodeToString(opID[:])
}

func ParseReplicaCount(raw string) int {
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return 1
	}
	return n
}
