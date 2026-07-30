package xdpstats

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

func WriteRedis(ctx context.Context, rdb redis.Cmdable, snap Snapshot) error {
	if rdb == nil {
		return nil
	}
	snap.UpdatedAt = snap.UpdatedAt.UTC()
	raw, err := json.Marshal(snap)
	if err != nil {
		return err
	}
	return rdb.Set(ctx, redisSnapshotKey, raw, 10*time.Minute).Err()
}

func ReadRedis(ctx context.Context, rdb redis.Cmdable) (Snapshot, error) {
	if rdb == nil {
		return Snapshot{}, fmt.Errorf("redis client is nil")
	}
	raw, err := rdb.Get(ctx, redisSnapshotKey).Bytes()
	if err != nil {
		return Snapshot{}, err
	}
	var snap Snapshot
	if err := json.Unmarshal(raw, &snap); err != nil {
		return Snapshot{}, err
	}
	return snap, nil
}
