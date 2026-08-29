package testutil

import (
	"path/filepath"
	"testing"

	"ad-event-processor/internal/domain/db"
	ingestion "ad-event-processor/internal/ingest"
)

func NewAdsRegistry(t testing.TB, repo db.Querier) *ingestion.Registry {
	t.Helper()
	r := ingestion.NewRegistry(repo)
	r.SetReplicaPath(filepath.Join(t.TempDir(), "campaigns_replica.json"))
	return r
}
