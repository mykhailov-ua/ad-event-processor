package controlplane

import (
	"testing"

	"ad-event-processor/internal/fraud"
	"github.com/stretchr/testify/require"
)

func TestNormalizeFraudExplainHours(t *testing.T) {
	require.Equal(t, 24, normalizeFraudExplainHours(0))
	require.Equal(t, 24, normalizeFraudExplainHours(-1))
	require.Equal(t, 168, normalizeFraudExplainHours(999))
	require.Equal(t, 12, normalizeFraudExplainHours(12))
}

func TestFraudFeatureMap_has16Dims(t *testing.T) {
	row := fraud.FeatureRow{
		Events:           10,
		Clicks:           2,
		SpendMicro:       1_000_000,
		BudgetLimitMicro: 10_000_000,
		UniqueUsers:      3,
		UniqueUAs:        4,
	}
	out := fraudFeatureMap(row)
	require.Len(t, out, len(fraud.FeatureNames))
	require.Equal(t, float64(10), out["events"])
	require.InDelta(t, 0.2, out["ctr"], 0.0001)
}

func TestDecideWithCampaign_replaySignals(t *testing.T) {
	row := fraud.FeatureRow{
		Events:           500,
		Clicks:           5,
		SpendMicro:       5_000_000,
		BudgetLimitMicro: 10_000_000,
		UniqueUsers:      40,
		UniqueUAs:        2,
	}
	decision := fraud.DecideWithCampaign(row, 0.15, 30, 60, 80, 100)
	require.NotEmpty(t, string(decision.Tier))
	require.GreaterOrEqual(t, decision.Score, 0)
}
