package reports

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFinalizeDataFreshness_staleLabel_holdout(t *testing.T) {
	t.Parallel()

	got := finalizeDataFreshness(DataFreshnessDTO{
		AsOf:         time.Now().UTC().Format(time.RFC3339),
		Consistency:  "eventual",
		Stale:        true,
		CHLagSeconds: 120,
	})
	require.Equal(t, "Analytics stale (120s lag)", got.FreshnessLabel)
	require.NotEmpty(t, got.AsOfDisplay)
}

func TestFinalizeDataFreshness_liveLabel_holdout(t *testing.T) {
	t.Parallel()

	got := finalizeDataFreshness(DataFreshnessDTO{
		AsOf:        "2026-09-11T08:00:00Z",
		Consistency: "strong",
		Stale:       false,
	})
	require.Contains(t, got.FreshnessLabel, "Updated")
}
