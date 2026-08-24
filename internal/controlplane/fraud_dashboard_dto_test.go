package controlplane_test

import (
	"encoding/json"
	"testing"

	"ad-event-processor/internal/controlplane"
	"github.com/stretchr/testify/require"
)

func TestFraudDashboardDTO_serialization(t *testing.T) {
	consistent := true
	dto := controlplane.FraudDashboardDTO{
		CustomerID:         "550e8400-e29b-41d4-a716-446655440000",
		MLActiveVersionID:  "v1",
		MLPrecision:        0.91,
		MLRecall:           0.42,
		MLDriftDetected:    true,
		MLDriftSummary:     "Traffic mix changed more than 30% vs training on one or more raw features.",
		MLEvalGeneratedAt:  "2026-02-01T10:00:00Z",
		MLEvalStatus:       "drift_detected",
		MLEvalStale:        true,
		MLLabelMethod:      "proxy",
		MLShardsConsistent: &consistent,
		FraudTierThresholds: controlplane.FraudTierThresholdsDTO{
			Scope:      "platform_default",
			PassMax:    30,
			SuspectMax: 60,
			IVTMax:     80,
			BlockAbove: 100,
		},
	}

	raw, err := json.Marshal(dto)
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(raw, &decoded))
	require.Equal(t, "proxy", decoded["ml_label_method"])
	require.Equal(t, "drift_detected", decoded["ml_eval_status"])
	require.Equal(t, true, decoded["ml_eval_stale"])
	require.Equal(t, "platform_default", decoded["fraud_tier_thresholds"].(map[string]any)["scope"])
}
