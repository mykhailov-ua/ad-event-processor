package testutil

import (
	"context"
	"path/filepath"
	"testing"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/ingestion"

	redis "github.com/redis/go-redis/v9"
)

type WatchingRegistry struct {
	*ingestion.Registry
}

func NewWatchingRegistry(t testing.TB, queries db.Querier, redisClient redis.UniversalClient, channel string) *WatchingRegistry {
	t.Helper()
	r := ingestion.NewRegistry(queries)
	r.SetReplicaPath(filepath.Join(t.TempDir(), "campaigns_replica.json"))
	r.StartWatch(context.Background(), redisClient, channel)
	return &WatchingRegistry{Registry: r}
}

func AttachBudgetCacheWarmer(registry *ingestion.Registry, redisShards []redis.UniversalClient, sharder domain.Sharder) {
	if registry == nil {
		return
	}
	registry.SetBudgetWarmer(ingestion.NewBudgetCacheWarmer(redisShards, sharder))
}
