package automation

import (
	"fmt"
	"strings"
)

const LicenseFeaturePlatformCampaignAPI = "ad_platform_campaign_api"

type PresetParameter struct {
	Key         string   `json:"key"`
	Type        string   `json:"type"`
	Description string   `json:"description,omitempty"`
	Default     float64  `json:"default,omitempty"`
	Min         *float64 `json:"min,omitempty"`
	Max         *float64 `json:"max,omitempty"`
}

type Preset struct {
	Key                     string            `json:"key"`
	Title                   string            `json:"title"`
	Description             string            `json:"description"`
	ParametersSchema        []PresetParameter `json:"parameters_schema"`
	RequiredLicenseFeatures []string          `json:"required_license_features,omitempty"`
}

type ExpandedRule struct {
	Name                string
	Metric              string
	Operator            string
	Threshold           float64
	WindowMinutes       int
	GroupBy             string
	Actions             []Action
	CooldownMinutes     int
	EvalIntervalMinutes int
}

type presetDef struct {
	Preset
	expand func(map[string]float64) (ExpandedRule, error)
}

func ListPresets() []Preset {
	defs := presetCatalog()
	out := make([]Preset, 0, len(defs))
	for _, d := range defs {
		out = append(out, d.Preset)
	}
	return out
}

func PresetKeys() []string {
	defs := presetCatalog()
	out := make([]string, 0, len(defs))
	for _, d := range defs {
		out = append(out, d.Key)
	}
	return out
}

func ExpandPreset(key string, params map[string]float64) (ExpandedRule, error) {
	key = strings.TrimSpace(key)
	for _, d := range presetCatalog() {
		if d.Key != key {
			continue
		}
		merged := mergePresetParams(d.ParametersSchema, params)
		return d.expand(merged)
	}
	return ExpandedRule{}, fmt.Errorf("unknown preset_key %q; valid keys: %s", key, strings.Join(PresetKeys(), ", "))
}

func mergePresetParams(schema []PresetParameter, overrides map[string]float64) map[string]float64 {
	out := make(map[string]float64, len(schema))
	for _, p := range schema {
		out[p.Key] = p.Default
	}
	for k, v := range overrides {
		out[k] = v
	}
	return out
}

func presetFloat(params map[string]float64, key string, def float64) float64 {
	if v, ok := params[key]; ok {
		return v
	}
	return def
}

func presetInt(params map[string]float64, key string, def int) int {
	return int(presetFloat(params, key, float64(def)))
}

func presetCatalog() []presetDef {
	return []presetDef{
		{
			Preset: Preset{
				Key:         "placement_roi_guard",
				Title:       "Placement ROI guard",
				Description: "Blacklist placement when ROI stays below threshold for the observation window.",
				ParametersSchema: []PresetParameter{
					{Key: "roi_threshold", Type: "number", Description: "ROI percent threshold", Default: -40, Max: ptrFloat(0)},
					{Key: "window_minutes", Type: "integer", Description: "Observation window minutes", Default: 30, Min: ptrFloat(15), Max: ptrFloat(1440)},
					{Key: "cooldown_minutes", Type: "integer", Description: "Cooldown after fire", Default: 60, Min: ptrFloat(15), Max: ptrFloat(10080)},
					{Key: "eval_interval_minutes", Type: "integer", Description: "Evaluation interval minutes", Default: 15, Min: ptrFloat(5), Max: ptrFloat(60)},
				},
			},
			expand: func(params map[string]float64) (ExpandedRule, error) {
				return ExpandedRule{
					Name:                "Placement ROI guard",
					Metric:              "roi_pct",
					Operator:            "lt",
					Threshold:           presetFloat(params, "roi_threshold", -40),
					WindowMinutes:       presetInt(params, "window_minutes", 30),
					GroupBy:             GroupByPlacement,
					Actions:             []Action{{Type: ActionBlacklistPlacement}},
					CooldownMinutes:     presetInt(params, "cooldown_minutes", 60),
					EvalIntervalMinutes: presetInt(params, "eval_interval_minutes", 15),
				}, nil
			},
		},
		{
			Preset: Preset{
				Key:         "fraud_rate_guard",
				Title:       "Fraud reject rate guard",
				Description: "Blacklist placement when hard fraud reject rate exceeds threshold.",
				ParametersSchema: []PresetParameter{
					{Key: "fraud_rate_threshold", Type: "number", Description: "Fraud reject rate percent", Default: 25, Min: ptrFloat(1), Max: ptrFloat(100)},
					{Key: "window_minutes", Type: "integer", Description: "Observation window minutes", Default: 30, Min: ptrFloat(15), Max: ptrFloat(1440)},
					{Key: "cooldown_minutes", Type: "integer", Description: "Cooldown after fire", Default: 60, Min: ptrFloat(15), Max: ptrFloat(10080)},
					{Key: "eval_interval_minutes", Type: "integer", Description: "Evaluation interval minutes", Default: 15, Min: ptrFloat(5), Max: ptrFloat(60)},
				},
			},
			expand: func(params map[string]float64) (ExpandedRule, error) {
				return ExpandedRule{
					Name:                "Fraud rate guard",
					Metric:              "fraud_reject_rate",
					Operator:            "gt",
					Threshold:           presetFloat(params, "fraud_rate_threshold", 25),
					WindowMinutes:       presetInt(params, "window_minutes", 30),
					GroupBy:             GroupByPlacement,
					Actions:             []Action{{Type: ActionBlacklistPlacement}},
					CooldownMinutes:     presetInt(params, "cooldown_minutes", 60),
					EvalIntervalMinutes: presetInt(params, "eval_interval_minutes", 15),
				}, nil
			},
		},
		{
			Preset: Preset{
				Key:         "spend_cap_guard",
				Title:       "Spend cap guard",
				Description: "Pause campaign when spend in micro-units exceeds threshold within the window.",
				ParametersSchema: []PresetParameter{
					{Key: "spend_threshold_micro", Type: "number", Description: "Spend threshold in micro-units", Default: 50_000_000, Min: ptrFloat(1)},
					{Key: "window_minutes", Type: "integer", Description: "Observation window minutes", Default: 60, Min: ptrFloat(15), Max: ptrFloat(1440)},
					{Key: "cooldown_minutes", Type: "integer", Description: "Cooldown after fire", Default: 60, Min: ptrFloat(15), Max: ptrFloat(10080)},
					{Key: "eval_interval_minutes", Type: "integer", Description: "Evaluation interval minutes", Default: 15, Min: ptrFloat(5), Max: ptrFloat(60)},
				},
			},
			expand: func(params map[string]float64) (ExpandedRule, error) {
				return ExpandedRule{
					Name:                "Spend cap guard",
					Metric:              "spend_micro",
					Operator:            "gt",
					Threshold:           presetFloat(params, "spend_threshold_micro", 50_000_000),
					WindowMinutes:       presetInt(params, "window_minutes", 60),
					GroupBy:             GroupByCampaign,
					Actions:             []Action{{Type: ActionPauseCampaign}},
					CooldownMinutes:     presetInt(params, "cooldown_minutes", 60),
					EvalIntervalMinutes: presetInt(params, "eval_interval_minutes", 15),
				}, nil
			},
		},
		{
			Preset: Preset{
				Key:         "silent_reject_spike",
				Title:       "Silent reject spike",
				Description: "Blacklist placement when silent reject rate exceeds threshold.",
				ParametersSchema: []PresetParameter{
					{Key: "silent_reject_threshold", Type: "number", Description: "Silent reject rate percent", Default: 15, Min: ptrFloat(1), Max: ptrFloat(100)},
					{Key: "window_minutes", Type: "integer", Description: "Observation window minutes", Default: 30, Min: ptrFloat(15), Max: ptrFloat(1440)},
					{Key: "cooldown_minutes", Type: "integer", Description: "Cooldown after fire", Default: 60, Min: ptrFloat(15), Max: ptrFloat(10080)},
					{Key: "eval_interval_minutes", Type: "integer", Description: "Evaluation interval minutes", Default: 15, Min: ptrFloat(5), Max: ptrFloat(60)},
				},
			},
			expand: func(params map[string]float64) (ExpandedRule, error) {
				return ExpandedRule{
					Name:                "Silent reject spike",
					Metric:              "silent_reject_rate",
					Operator:            "gt",
					Threshold:           presetFloat(params, "silent_reject_threshold", 15),
					WindowMinutes:       presetInt(params, "window_minutes", 30),
					GroupBy:             GroupByPlacement,
					Actions:             []Action{{Type: ActionBlacklistPlacement}},
					CooldownMinutes:     presetInt(params, "cooldown_minutes", 60),
					EvalIntervalMinutes: presetInt(params, "eval_interval_minutes", 15),
				}, nil
			},
		},
	}
}

func ptrFloat(v float64) *float64 {
	return &v
}

func PresetUsesPlatformPause(key string) bool {
	expanded, err := ExpandPreset(key, nil)
	if err != nil {
		return false
	}
	for _, action := range expanded.Actions {
		if action.Type == ActionPlatformPause {
			return true
		}
	}
	return false
}
