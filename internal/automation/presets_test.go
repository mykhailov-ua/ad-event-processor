package automation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpandPreset_placementRoiGuard_matchesManualRule(t *testing.T) {
	expanded, err := ExpandPreset("placement_roi_guard", map[string]float64{
		"roi_threshold":         -40,
		"window_minutes":        30,
		"cooldown_minutes":      60,
		"eval_interval_minutes": 15,
	})
	require.NoError(t, err)
	manual := ExpandedRule{
		Name:                "Placement ROI guard",
		Metric:              "roi_pct",
		Operator:            "lt",
		Threshold:           -40,
		WindowMinutes:       30,
		GroupBy:             GroupByPlacement,
		Actions:             []Action{{Type: ActionBlacklistPlacement}},
		CooldownMinutes:     60,
		EvalIntervalMinutes: 15,
	}
	assert.Equal(t, manual, expanded)
}

func TestExpandPreset_unknownKey_holdout(t *testing.T) {
	_, err := ExpandPreset("missing_preset", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "placement_roi_guard")
}

func TestPresetCatalog_noPlatformPause_holdout(t *testing.T) {
	for _, key := range PresetKeys() {
		assert.False(t, PresetUsesPlatformPause(key), "preset %s must not enable platform_pause", key)
	}
}

func TestListPresets_containsBundledKeys(t *testing.T) {
	presets := ListPresets()
	keys := make(map[string]struct{}, len(presets))
	for _, p := range presets {
		keys[p.Key] = struct{}{}
	}
	for _, want := range []string{"placement_roi_guard", "fraud_rate_guard", "spend_cap_guard", "silent_reject_spike"} {
		_, ok := keys[want]
		assert.True(t, ok, "missing preset %s", want)
	}
}
