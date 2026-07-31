package testutil

import (
	"espx/internal/config"
	"espx/internal/ingestion"

	redis "github.com/redis/go-redis/v9"
)

func NewSettingsWatcher(rdbs []redis.UniversalClient, cfg *config.Config) *ingestion.SettingsWatcher {
	return ingestion.NewSettingsWatcher(rdbs, cfg)
}

func NewEmergencyBreakerFilter(watcher *ingestion.SettingsWatcher) *ingestion.EmergencyBreakerFilter {
	return ingestion.NewEmergencyBreakerFilter(watcher)
}
