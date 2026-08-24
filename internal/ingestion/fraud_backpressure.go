package ingestion

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"ad-event-processor/internal/metrics"

	redis "github.com/redis/go-redis/v9"
)

const fraudAggForceKey = "fraud:agg_force"

type FraudBackpressureConfig struct {
	RedisShards []redis.UniversalClient
	Writer      *FraudStreamWriter
	Stream      string
	EventStream string
	Group       string
	LagSec      int
	Interval    time.Duration
}

func StartFraudBackpressureWatcher(ctx context.Context, cfg FraudBackpressureConfig) {
	if cfg.Writer == nil || len(cfg.RedisShards) == 0 {
		return
	}
	if cfg.Interval <= 0 {
		cfg.Interval = 2 * time.Second
	}
	if cfg.LagSec <= 0 {
		cfg.LagSec = 60
	}
	go func() {
		ticker := time.NewTicker(cfg.Interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				force := readFraudAggForce(ctx, cfg.RedisShards)
				cfg.Writer.SetForceAggregate(force)
				publishStreamPELAges(ctx, cfg)
			}
		}
	}()
}

func readFraudAggForce(ctx context.Context, redisShards []redis.UniversalClient) bool {
	for _, redisClient := range redisShards {
		if redisClient == nil {
			continue
		}
		v, err := redisClient.Get(ctx, fraudAggForceKey).Result()
		if err != nil {
			continue
		}
		if v == "1" || v == "true" {
			return true
		}
	}
	return false
}

func PublishFraudConsumerLag(ctx context.Context, redisClient redis.UniversalClient, stream, group string, lagSec int) {
	if redisClient == nil || stream == "" || group == "" || lagSec <= 0 {
		return
	}
	age := oldestPELIdleSeconds(ctx, redisClient, stream, group)
	if age < 0 {
		return
	}
	force := age > float64(lagSec)
	val := "0"
	ttl := time.Duration(lagSec) * time.Second
	if force {
		val = "1"
		ttl = 2 * time.Duration(lagSec) * time.Second
	}
	if err := redisClient.Set(ctx, fraudAggForceKey, val, ttl).Err(); err != nil {
		slog.Debug("fraud agg force publish failed", "error", err)
	}
}

func oldestPELIdleSeconds(ctx context.Context, redisClient redis.UniversalClient, stream, group string) float64 {
	pending, err := redisClient.XPendingExt(ctx, &redis.XPendingExtArgs{
		Stream: stream,
		Group:  group,
		Start:  "-",
		End:    "+",
		Count:  1,
	}).Result()
	if err != nil || len(pending) == 0 {
		return -1
	}
	return pending[0].Idle.Seconds()
}

func publishStreamPELAges(ctx context.Context, cfg FraudBackpressureConfig) {
	streams := []string{}
	if cfg.EventStream != "" {
		streams = append(streams, cfg.EventStream)
	}
	if cfg.Stream != "" {
		streams = append(streams, cfg.Stream)
	}
	group := cfg.Group
	if group == "" {
		group = "ad_event_processor"
	}
	for i, redisClient := range cfg.RedisShards {
		if redisClient == nil {
			continue
		}
		shard := strconv.Itoa(i)
		for _, stream := range streams {
			age := oldestPELIdleSeconds(ctx, redisClient, stream, group)
			if age < 0 {
				age = 0
			}
			metrics.FraudStreamPELAgeSeconds.WithLabelValues(stream, shard).Set(age)
		}
	}
}

func StartFraudLagPublisher(ctx context.Context, redisShards []redis.UniversalClient, stream, group string, lagSec int, interval time.Duration) {
	if len(redisShards) == 0 || stream == "" {
		return
	}
	if interval <= 0 {
		interval = 2 * time.Second
	}
	if lagSec <= 0 {
		lagSec = 60
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				for _, redisClient := range redisShards {
					if redisClient == nil {
						continue
					}
					PublishFraudConsumerLag(ctx, redisClient, stream, group, lagSec)
					break
				}
			}
		}
	}()
}
