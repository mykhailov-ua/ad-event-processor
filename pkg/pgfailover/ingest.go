package pgfailover

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type IngestRuntime struct {
	mu          sync.Mutex
	subscribers []*Subscriber
}

func (r *IngestRuntime) Stop() {
	if r == nil {
		return
	}
	for _, sub := range r.subscribers {
		if sub != nil {
			sub.Stop()
		}
	}
}

func (r *IngestRuntime) CurrentDSN() string {
	if r == nil {
		return ""
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, sub := range r.subscribers {
		if sub != nil {
			if dsn := sub.CurrentDSN(); dsn != "" {
				return dsn
			}
		}
	}
	return ""
}

type IngestSubscriberConfig struct {
	MaxConns int
	MinConns int
	Interval time.Duration
}

func StartIngestSubscribers(
	ctx context.Context,
	redisShards []redis.UniversalClient,
	cfg IngestSubscriberConfig,
	onReconnect ReconnectFunc,
) *IngestRuntime {
	if onReconnect == nil {
		return nil
	}
	primary := firstConnectedRedis(redisShards)
	if primary == nil {
		slog.Warn("pg failover ingest disabled: no connected redis shard for DSN subscription")
		return nil
	}
	if cfg.Interval <= 0 {
		cfg.Interval = time.Second
	}
	if cfg.MaxConns <= 0 {
		cfg.MaxConns = 4
	}
	if cfg.MinConns <= 0 {
		cfg.MinConns = 1
	}
	gate := NewFencingGate(primary)
	subCfg := SubscriberConfig(cfg)
	rt := &IngestRuntime{}
	sub := NewSubscriber(primary, gate, onReconnect, subCfg)
	sub.Start(ctx)
	rt.subscribers = append(rt.subscribers, sub)
	for _, redisClient := range redisShards {
		if redisClient == nil || redisClient == primary {
			continue
		}
		extra := NewSubscriber(redisClient, gate, onReconnect, subCfg)
		extra.Start(ctx)
		rt.subscribers = append(rt.subscribers, extra)
	}
	slog.Info("pg failover ingest subscriber started", "max_conns", cfg.MaxConns, "shards", len(rt.subscribers))
	return rt
}

func firstConnectedRedis(redisShards []redis.UniversalClient) redis.UniversalClient {
	for _, redisClient := range redisShards {
		if redisClient != nil {
			return redisClient
		}
	}
	return nil
}
