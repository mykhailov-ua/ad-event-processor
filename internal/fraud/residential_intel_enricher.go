package fraud

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"ad-event-processor/internal/edge"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/redis/go-redis/v9"
)

type ResidentialIntelEnricher struct {
	provider    ResidentialIntelProvider
	cache       *ResidentialIntelCache
	chWrite     driver.Conn
	redisClient redis.Cmdable
	feedDir     string
	providerID  string
	recentLim   int
	batchLim    int
	interval    time.Duration
}

type ResidentialIntelEnricherConfig struct {
	Provider    ResidentialIntelProvider
	Cache       *ResidentialIntelCache
	CHWrite     driver.Conn
	RedisClient redis.Cmdable
	FeedDir     string
	ProviderID  string
	RecentLim   int
	BatchLim    int
	Interval    time.Duration
}

func NewResidentialIntelEnricher(cfg ResidentialIntelEnricherConfig) *ResidentialIntelEnricher {
	if cfg.Provider == nil || cfg.Cache == nil {
		return nil
	}
	recent := cfg.RecentLim
	if recent <= 0 {
		recent = 128
	}
	batch := cfg.BatchLim
	if batch <= 0 {
		batch = 32
	}
	interval := cfg.Interval
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	providerID := cfg.ProviderID
	if providerID == "" {
		providerID = "http"
	}
	return &ResidentialIntelEnricher{
		provider:    cfg.Provider,
		cache:       cfg.Cache,
		chWrite:     cfg.CHWrite,
		redisClient: cfg.RedisClient,
		feedDir:     cfg.FeedDir,
		providerID:  providerID,
		recentLim:   recent,
		batchLim:    batch,
		interval:    interval,
	}
}

func (e *ResidentialIntelEnricher) RunLoop(ctx context.Context) error {
	if e == nil {
		return fmt.Errorf("residential intel enricher: nil receiver")
	}
	ticker := time.NewTicker(e.interval)
	defer ticker.Stop()

	if _, err := e.Run(ctx); err != nil && ctx.Err() == nil {
		slog.Warn("residential intel enricher initial cycle failed", "error", err)
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			stats, err := e.Run(ctx)
			if err != nil && ctx.Err() == nil {
				slog.Warn("residential intel enricher cycle failed", "error", err)
				continue
			}
			if stats.LookedUp > 0 || stats.FeedAppended > 0 {
				slog.Info("residential intel enricher cycle complete",
					"looked_up", stats.LookedUp,
					"cached_hits", stats.CacheHits,
					"feed_appended", stats.FeedAppended,
				)
			}
		}
	}
}

type ResidentialIntelRunStats struct {
	LookedUp     int
	CacheHits    int
	FeedAppended int
}

func (e *ResidentialIntelEnricher) Run(ctx context.Context) (ResidentialIntelRunStats, error) {
	var stats ResidentialIntelRunStats
	if e == nil {
		return stats, fmt.Errorf("residential intel enricher: nil receiver")
	}
	if e.redisClient == nil {
		return stats, nil
	}

	entries, err := edge.ListRecent(ctx, e.redisClient, e.recentLim)
	if err != nil {
		residentialIntelErrorsTotal.Inc()
		return stats, fmt.Errorf("list edge IPs: %w", err)
	}
	if len(entries) == 0 {
		return stats, nil
	}

	lookups := 0
	for _, entry := range entries {
		if lookups >= e.batchLim {
			break
		}
		ip := entry.IP
		if ip == "" {
			continue
		}

		cached, hit, cacheErr := e.cache.Get(ctx, ip)
		if cacheErr != nil {
			residentialIntelErrorsTotal.Inc()
			return stats, cacheErr
		}
		result := cached
		freshLookup := false
		if hit {
			stats.CacheHits++
		} else {
			result, err = e.provider.Lookup(ctx, ip)
			if err != nil {
				residentialIntelErrorsTotal.Inc()
				slog.Warn("residential intel provider lookup failed", "ip", ip, "error", err)
				continue
			}
			if setErr := e.cache.Set(ctx, ip, result); setErr != nil {
				residentialIntelErrorsTotal.Inc()
				return stats, setErr
			}
			freshLookup = true
			lookups++
			stats.LookedUp++
			residentialIntelLookupsTotal.Inc()
			now := time.Now().UTC()
			if chErr := insertResidentialIntelCH(ctx, e.chWrite, ip, result, e.providerID, now); chErr != nil {
				residentialIntelErrorsTotal.Inc()
				slog.Warn("residential intel clickhouse insert failed", "ip", ip, "error", chErr)
			}
		}

		if !freshLookup || !result.IsResidentialProxyFarm() {
			continue
		}
		if feedErr := appendResidentialIntelFeed(e.feedDir, ip); feedErr != nil {
			if errors.Is(feedErr, ErrInvalidIP) {
				continue
			}
			residentialIntelErrorsTotal.Inc()
			slog.Warn("residential intel feed append failed", "ip", ip, "error", feedErr)
			continue
		}
		stats.FeedAppended++
		residentialIntelFeedAppendedTotal.Inc()
	}

	return stats, nil
}
