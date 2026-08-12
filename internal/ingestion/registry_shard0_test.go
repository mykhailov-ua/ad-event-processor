package ingestion

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/bidshard/ad-event-processor/internal/config"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistry_BootstrapFromReplica(t *testing.T) {
	replicaPath := filepath.Join(t.TempDir(), "campaigns_replica.json")

	id1 := uuid.New()
	mock := &MockRepo{ids: []pgtype.UUID{{Bytes: id1, Valid: true}}}
	r1 := NewRegistry(mock)
	r1.SetReplicaPath(replicaPath)
	_, err := r1.Sync(context.Background())
	require.NoError(t, err)

	r2 := NewRegistry(&MockRepo{err: errors.New("pg down")})
	r2.SetReplicaPath(replicaPath)

	n, err := r2.BootstrapFromReplica()
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	assert.True(t, r2.Exists(id1))

	_, err = os.Stat(replicaPath)
	require.NoError(t, err)
}

func TestSettingsWatcher_trySyncFromPGWhenStale(t *testing.T) {
	cfg := &config.Config{
		RateLimitPerMin:  100,
		ClickAmount:      100_000,
		ImpressionAmount: 10_000,
	}
	sw := NewSettingsWatcher(nil, cfg)
	sw.SetPGFallback(
		func(context.Context) (map[string]string, int64, error) {
			return map[string]string{"rate_limit_per_min": "250"}, 9, nil
		},
		func() bool { return true },
	)
	sw.trySyncFromPG(context.Background())
	assert.Equal(t, 250, sw.Get().RateLimitPerMin)
	assert.Equal(t, int64(9), sw.currentVersion)
}

func TestSettingsWatcher_trySyncFromPG_skipsWhenFresh(t *testing.T) {
	cfg := &config.Config{RateLimitPerMin: 100}
	sw := NewSettingsWatcher(nil, cfg)
	sw.SetPGFallback(
		func(context.Context) (map[string]string, int64, error) {
			t.Fatal("pg sync must not run when pub/sub healthy")
			return nil, 0, nil
		},
		func() bool { return false },
	)
	sw.trySyncFromPG(context.Background())
}
