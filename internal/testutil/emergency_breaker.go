package testutil

import (
	"ad-event-processor/internal/config"
	"ad-event-processor/internal/ingestion"

	redis "github.com/redis/go-redis/v9"
)

func NewSettingsWatcher(redisShards []redis.UniversalClient, cfg *config.Config) *ingestion.SettingsWatcher {
	return ingestion.NewSettingsWatcher(redisShards, cfg)
}

func NewEmergencyBreakerFilter(watcher *ingestion.SettingsWatcher) *ingestion.EmergencyBreakerFilter {
	return ingestion.NewEmergencyBreakerFilter(watcher)
}
