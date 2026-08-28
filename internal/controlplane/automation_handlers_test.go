package controlplane

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ad-event-processor/internal/automation"
	"ad-event-processor/internal/config"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAutomationHandlers_listPresets(t *testing.T) {
	h := &automation.HTTPHandlers{Rules: &automation.RulesService{}}
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/automation/presets", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var presets []automation.Preset
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &presets))
	require.NotEmpty(t, presets)
}

func TestCreateAutomationRule_evalIntervalBelowFloor_holdout(t *testing.T) {
	svc := &Service{cfg: &config.Config{}}
	svc.cfg.Management.AutomationRulesIntervalMin = 15
	req := automation.UpsertRuleRequest{
		CustomerID:          uuid.New().String(),
		Name:                "fast",
		Metric:              "roi_pct",
		Operator:            "lt",
		Threshold:           -10,
		WindowMinutes:       30,
		GroupBy:             automation.GroupByPlacement,
		Actions:             []automation.Action{{Type: automation.ActionBlacklistPlacement}},
		EvalIntervalMinutes: 5,
		Enabled:             true,
	}
	_, err := svc.AutomationRules().BuildRuleParams(context.Background(), req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least 15")
}

func TestApplyAutomationPreset_expandsPlacementRoiGuard(t *testing.T) {
	req := automation.UpsertRuleRequest{
		CustomerID: uuid.New().String(),
		PresetKey:  "placement_roi_guard",
		PresetParameters: map[string]float64{
			"roi_threshold": -50,
		},
		Enabled: true,
	}
	out, err := automation.ApplyPreset(req)
	require.NoError(t, err)
	assert.Equal(t, "placement_roi_guard", out.PresetKey)
	assert.Equal(t, "roi_pct", out.Metric)
	assert.Equal(t, "lt", out.Operator)
	assert.Equal(t, -50.0, out.Threshold)
	assert.Equal(t, automation.GroupByPlacement, out.GroupBy)
	require.Len(t, out.Actions, 1)
	assert.Equal(t, automation.ActionBlacklistPlacement, out.Actions[0].Type)
}

func TestApplyAutomationPreset_unknownPreset_holdout(t *testing.T) {
	_, err := automation.ApplyPreset(automation.UpsertRuleRequest{PresetKey: "nope"})
	require.Error(t, err)
}
