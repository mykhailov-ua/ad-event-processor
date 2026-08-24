package ingestion

import (
	"context"
	"strings"
	"time"

	"ad-event-processor/internal/domain"
)

func parseBlacklistUpdatePayload(payload string) (ip, reason string, ok bool) {
	idx := strings.LastIndex(payload, ":")
	if idx <= 0 || idx >= len(payload)-1 {
		return "", "", false
	}
	return payload[:idx], payload[idx+1:], true
}

func (f *FraudBlacklistFilter) InvalidateIP(ip string) {
	if f == nil || ip == "" {
		return
	}
	shard := &f.shards[fraudBlacklistShardIndex(ip)]
	shard.mu.Lock()
	delete(shard.m, ip)
	shard.mu.Unlock()
}

func (f *FraudBlacklistFilter) RunInvalidationSubscriber(ctx context.Context, channel string) {
	if f == nil {
		return
	}
	channel = domain.DefaultBlacklistUpdateChannel(channel)
	backoff := time.Second
	for ctx.Err() == nil {
		if !f.consumeInvalidationUpdates(ctx, channel) {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}

func (f *FraudBlacklistFilter) consumeInvalidationUpdates(ctx context.Context, channel string) bool {
	redisClient := pickLocalGlobalShard(f.redisShards)
	if redisClient == nil {
		return true
	}

	pubsub := redisClient.Subscribe(ctx, channel)
	defer func() { _ = pubsub.Close() }()

	for msg := range pubsub.Channel() {
		if ctx.Err() != nil {
			return false
		}
		ip, reason, ok := parseBlacklistUpdatePayload(msg.Payload)
		if !ok || reason != "fraud" {
			continue
		}
		f.InvalidateIP(ip)
	}
	return true
}
