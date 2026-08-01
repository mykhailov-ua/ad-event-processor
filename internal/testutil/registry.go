package testutil

import (
	"path/filepath"
	"testing"

	"espx/internal/domain/db"
	"espx/internal/ingestion"
)

func NewAdsRegistry(t testing.TB, repo db.Querier) *ingestion.Registry {
	t.Helper()
	r := ingestion.NewRegistry(repo)
	r.SetReplicaPath(filepath.Join(t.TempDir(), "campaigns_replica.json"))
	return r
}
