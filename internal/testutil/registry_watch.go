package testutil

import (
	"context"
	"path/filepath"
	"testing"

	"espx/internal/domain"
	"espx/internal/domain/db"
	"espx/internal/ingestion"

	redis "github.com/redis/go-redis/v9"
)

type WatchingRegistry struct {
	*ingestion.Registry
}

func NewWatchingRegistry(t testing.TB, queries db.Querier, rdb redis.UniversalClient, channel string) *WatchingRegistry {
	t.Helper()
	r := ingestion.NewRegistry(queries)
	r.SetReplicaPath(filepath.Join(t.TempDir(), "campaigns_replica.json"))
	r.StartWatch(context.Background(), rdb, channel)
	return &WatchingRegistry{Registry: r}
}

func AttachBudgetCacheWarmer(registry *ingestion.Registry, rdbs []redis.UniversalClient, sharder domain.Sharder) {
	if registry == nil {
		return
	}
	registry.SetBudgetWarmer(ingestion.NewBudgetCacheWarmer(rdbs, sharder))
}
