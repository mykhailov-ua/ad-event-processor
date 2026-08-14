package pgfailover

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// IngestRuntime subscribes tracker/processor binaries to global PG DSN updates in Redis.
// Reconnect uses a hard pool swap: onReconnect receives a new pool, callers rewire dependents,
// then the previous pool is closed. In-flight queries on the old pool may fail until swap
// completes; /track does not use per-request PG fencing.
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

// StartIngestSubscribers mirrors control-plane PG failover subscription on ingest binaries.
func StartIngestSubscribers(
	ctx context.Context,
	rdbs []redis.UniversalClient,
	cfg IngestSubscriberConfig,
	onReconnect ReconnectFunc,
) *IngestRuntime {
	if onReconnect == nil {
		return nil
	}
	primary := firstConnectedRedis(rdbs)
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
	subCfg := SubscriberConfig{
		MaxConns: cfg.MaxConns,
		MinConns: cfg.MinConns,
		Interval: cfg.Interval,
	}
	rt := &IngestRuntime{}
	sub := NewSubscriber(primary, gate, onReconnect, subCfg)
	sub.Start(ctx)
	rt.subscribers = append(rt.subscribers, sub)
	for _, rdb := range rdbs {
		if rdb == nil || rdb == primary {
			continue
		}
		extra := NewSubscriber(rdb, gate, onReconnect, subCfg)
		extra.Start(ctx)
		rt.subscribers = append(rt.subscribers, extra)
	}
	slog.Info("pg failover ingest subscriber started", "max_conns", cfg.MaxConns, "shards", len(rt.subscribers))
	return rt
}

func firstConnectedRedis(rdbs []redis.UniversalClient) redis.UniversalClient {
	for _, rdb := range rdbs {
		if rdb != nil {
			return rdb
		}
	}
	return nil
}
