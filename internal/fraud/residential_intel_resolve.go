package fraud

import (
	"time"

	"github.com/bidshard/ad-event-processor/internal/config"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/redis/go-redis/v9"
)

func NewResidentialIntelEnricherFromConfig(cfg *config.Config, rdb redis.Cmdable, chWrite driver.Conn) (*ResidentialIntelEnricher, error) {
	if cfg == nil || !cfg.ExternalResidentialIntelRuntimeEnabled() {
		return nil, nil
	}
	if rdb == nil {
		return nil, nil
	}
	provider, err := NewHTTPResidentialIntelProvider(
		cfg.ExternalResidentialIntel.ProviderURL,
		string(cfg.ExternalResidentialIntel.APIKey),
		10*time.Second,
	)
	if err != nil {
		return nil, err
	}
	cache := NewResidentialIntelCache(rdb, cfg.ExternalResidentialIntel.CacheTTL)
	return NewResidentialIntelEnricher(ResidentialIntelEnricherConfig{
		Provider:   provider,
		Cache:      cache,
		CHWrite:    chWrite,
		RDB:        rdb,
		FeedDir:    cfg.ExternalResidentialIntel.FeedDir,
		ProviderID: cfg.ExternalResidentialIntel.ProviderLabel,
		RecentLim:  cfg.ExternalResidentialIntel.RecentLimit,
		BatchLim:   cfg.ExternalResidentialIntel.BatchSize,
		Interval:   cfg.ExternalResidentialIntel.ScanInterval,
	}), nil
}
