package edge

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const redisSnapshotKey = "edge:xdp:stats_snapshot"

type Snapshot struct {
	UpdatedAt    time.Time         `json:"updated_at"`
	Pass         uint64            `json:"pass"`
	PassAllow    uint64            `json:"pass_allowlist"`
	Drops        map[string]uint64 `json:"drops"`
	Fingerprints uint64            `json:"fingerprints"`
}

func WriteRedis(ctx context.Context, redisClient redis.Cmdable, snap Snapshot) error {
	if redisClient == nil {
		return nil
	}
	snap.UpdatedAt = snap.UpdatedAt.UTC()
	raw, err := json.Marshal(snap)
	if err != nil {
		return err
	}
	return redisClient.Set(ctx, redisSnapshotKey, raw, 10*time.Minute).Err()
}

func ReadRedis(ctx context.Context, redisClient redis.Cmdable) (Snapshot, error) {
	if redisClient == nil {
		return Snapshot{}, fmt.Errorf("redis client is nil")
	}
	raw, err := redisClient.Get(ctx, redisSnapshotKey).Bytes()
	if err != nil {
		return Snapshot{}, err
	}
	var snap Snapshot
	if err := json.Unmarshal(raw, &snap); err != nil {
		return Snapshot{}, err
	}
	return snap, nil
}

func ReadRedisAny(ctx context.Context, redisShards []redis.UniversalClient) (Snapshot, error) {
	if len(redisShards) == 0 {
		return Snapshot{}, fmt.Errorf("no redis client available")
	}
	var lastErr error
	for i, redisClient := range redisShards {
		if redisClient == nil {
			continue
		}
		snap, err := ReadRedis(ctx, redisClient)
		if err == nil {
			return snap, nil
		}
		lastErr = err
		_ = i
	}
	if lastErr != nil {
		return Snapshot{}, lastErr
	}
	return Snapshot{}, fmt.Errorf("no connected redis shard")
}
