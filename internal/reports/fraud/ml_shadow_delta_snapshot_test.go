package fraud

import (
	"testing"
	"time"

	"ad-event-processor/internal/reports"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMLShadowDeltaSnapshotFreshness_staleAfter24h_holdout(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)
	fresh := MLShadowDeltaSnapshotFreshness(reports.MLShadowDeltaSnapshot{
		GeneratedAt: now.Add(-12 * time.Hour),
	}, now)
	assert.False(t, fresh.Stale)
	assert.Equal(t, "snapshot", fresh.Consistency)

	stale := MLShadowDeltaSnapshotFreshness(reports.MLShadowDeltaSnapshot{
		GeneratedAt: now.Add(-25 * time.Hour),
	}, now)
	assert.True(t, stale.Stale)
}

func TestPaginateMLShadowDeltaSnapshotRows(t *testing.T) {
	t.Parallel()
	rows := []map[string]any{
		{"bucket": "a"},
		{"bucket": "b"},
		{"bucket": "c"},
	}
	page, total := PaginateMLShadowDeltaSnapshotRows(rows, 2, 1)
	require.Equal(t, int64(3), total)
	require.Len(t, page, 2)
	assert.Equal(t, "b", page[0]["bucket"])
}
