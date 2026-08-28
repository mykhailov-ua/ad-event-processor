package notify

import (
	"time"

	"ad-event-processor/internal/config"
)

type ServiceOptions struct {
	DedupCooldownSec           int64
	ClaimStaleSec              int64
	GroupParallelism           int
	RateLimitPerMinute         int
	TelegramRateLimitPerMinute int
}

func defaultServiceOptions() ServiceOptions {
	return ServiceOptions{
		DedupCooldownSec:           300,
		ClaimStaleSec:              300,
		GroupParallelism:           2,
		RateLimitPerMinute:         60,
		TelegramRateLimitPerMinute: 20,
	}
}

func (o ServiceOptions) dedupCooldown() time.Duration {
	if o.DedupCooldownSec <= 0 {
		return 5 * time.Minute
	}
	return time.Duration(o.DedupCooldownSec) * time.Second
}

func (o ServiceOptions) claimStale() time.Duration {
	if o.ClaimStaleSec <= 0 {
		return 5 * time.Minute
	}
	return time.Duration(o.ClaimStaleSec) * time.Second
}

func (o ServiceOptions) groupParallelism() int {
	if o.GroupParallelism <= 0 {
		return 1
	}
	return o.GroupParallelism
}

func ServiceOptionsFromConfig(cfg *config.Config) ServiceOptions {
	if cfg == nil {
		return defaultServiceOptions()
	}
	n := cfg.Notifier
	return ServiceOptions{
		DedupCooldownSec:           int64(n.DedupCooldownSec),
		ClaimStaleSec:              int64(n.ClaimStaleSec),
		GroupParallelism:           n.GroupParallelism,
		RateLimitPerMinute:         n.RateLimitPerMinute,
		TelegramRateLimitPerMinute: n.TelegramRateLimitPerMinute,
	}
}
