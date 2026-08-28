package fraudadmin

import (
	"testing"

	"ad-event-processor/internal/fraud"

	"github.com/stretchr/testify/require"
)

func TestCountFraudPreviewTiers(t *testing.T) {
	thresholds := campaignFraudThresholds{pass: 30, suspect: 60, ivt: 80, block: 100}
	scores := []float64{0.1, 0.5, 0.7, 0.95}
	counts, affected := countFraudPreviewTiers(scores, thresholds)
	require.Equal(t, int64(3), affected)
	require.Equal(t, int64(1), counts.Suspect)
	require.Equal(t, int64(1), counts.IVT)
	require.Equal(t, int64(1), counts.Block)
}

func TestMapProbabilityTierWithThresholds_matchesFraudPackage(t *testing.T) {
	tier, score := fraud.MapProbabilityTierWithThresholds(0.75, 30, 60, 80, 100)
	require.Equal(t, fraud.FraudTierIVT, tier)
	require.Greater(t, score, 60)
}
