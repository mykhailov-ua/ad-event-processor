package pgfailover

import (
	"context"
	"errors"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	redisFencingEpochKey = "espx:pg:global:fencing_epoch"
	redisActiveDSNKey    = "espx:pg:global:dsn"
	redisDSNEpochKey     = "espx:pg:global:dsn_epoch"
	redisNotifyChannel   = "espx:pg:global:notify"
)

var ErrStalePgFencingEpoch = errors.New("stale pg fencing epoch")

type FencingGate struct {
	rdb   redis.UniversalClient
	floor atomic.Uint64
}

func NewFencingGate(rdb redis.UniversalClient) *FencingGate {
	return &FencingGate{rdb: rdb}
}

func (g *FencingGate) Floor() uint64 {
	return g.floor.Load()
}

func (g *FencingGate) Refresh(ctx context.Context) error {
	if g == nil || g.rdb == nil {
		return nil
	}
	val, err := g.rdb.Get(ctx, redisFencingEpochKey).Result()
	if err == redis.Nil {
		return nil
	}
	if err != nil {
		return err
	}
	epoch, err := strconv.ParseUint(val, 10, 64)
	if err != nil {
		return err
	}
	for {
		cur := g.floor.Load()
		if epoch <= cur {
			return nil
		}
		if g.floor.CompareAndSwap(cur, epoch) {
			return nil
		}
	}
}

func (g *FencingGate) Validate(epoch uint64) error {
	if g == nil || epoch == 0 {
		return nil
	}
	if epoch < g.floor.Load() {
		return ErrStalePgFencingEpoch
	}
	return nil
}

func (g *FencingGate) AdvanceFloor(epoch uint64) {
	if g == nil || epoch == 0 {
		return
	}
	for {
		cur := g.floor.Load()
		if epoch <= cur {
			return
		}
		if g.floor.CompareAndSwap(cur, epoch) {
			return
		}
	}
}

func BumpEpoch(ctx context.Context, rdb redis.UniversalClient) (uint64, error) {
	epoch, err := rdb.Incr(ctx, redisFencingEpochKey).Result()
	if err != nil {
		return 0, err
	}
	return uint64(epoch), nil
}

func PublishDSN(ctx context.Context, rdb redis.UniversalClient, dsn string, fencingEpoch uint64) error {
	pipe := rdb.Pipeline()
	pipe.Set(ctx, redisActiveDSNKey, dsn, 0)
	pipe.Set(ctx, redisDSNEpochKey, strconv.FormatUint(fencingEpoch, 10), 0)
	pipe.Set(ctx, redisFencingEpochKey, strconv.FormatUint(fencingEpoch, 10), 0)
	if _, err := pipe.Exec(ctx); err != nil {
		return err
	}
	return rdb.Publish(ctx, redisNotifyChannel, dsn).Err()
}

func ActiveDSN(ctx context.Context, rdb redis.UniversalClient) (dsn string, epoch uint64, err error) {
	pipe := rdb.Pipeline()
	dsnCmd := pipe.Get(ctx, redisActiveDSNKey)
	epochCmd := pipe.Get(ctx, redisDSNEpochKey)
	if _, err = pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		return "", 0, err
	}
	dsn, err = dsnCmd.Result()
	if err != nil {
		return "", 0, err
	}
	raw, err := epochCmd.Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return "", 0, err
	}
	if raw != "" {
		parsed, parseErr := strconv.ParseUint(raw, 10, 64)
		if parseErr != nil {
			return "", 0, parseErr
		}
		epoch = parsed
	}
	return dsn, epoch, nil
}

func NotifyChannel() string {
	return redisNotifyChannel
}

func WaitForDSN(ctx context.Context, rdb redis.UniversalClient, wantDSN string, interval time.Duration) error {
	if interval <= 0 {
		interval = 200 * time.Millisecond
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		dsn, _, err := ActiveDSN(ctx, rdb)
		if err == nil && dsn == wantDSN {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
