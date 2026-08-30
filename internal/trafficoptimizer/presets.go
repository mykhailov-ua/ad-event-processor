package trafficoptimizer

import (
	"fmt"
	"strings"
)

type PresetParameter struct {
	Key         string   `json:"key"`
	Type        string   `json:"type"`
	Description string   `json:"description,omitempty"`
	Default     float64  `json:"default,omitempty"`
	Min         *float64 `json:"min,omitempty"`
	Max         *float64 `json:"max,omitempty"`
}

type Preset struct {
	Key              string            `json:"key"`
	Title            string            `json:"title"`
	Description      string            `json:"description"`
	ParametersSchema []PresetParameter `json:"parameters_schema"`
}

type ExpandedRule struct {
	Name                string
	Scope               string
	Objective           string
	Algorithm           string
	LookbackMinutes     int
	MinClicks           int
	MinConversions      int
	MinSpendMicro       int64
	EvalIntervalMinutes int
	CooldownMinutes     int
	MaxWeightDeltaPct   int
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
				Key:         "cr_best_performer",
				Title:       "CR best performer",
				Description: "Shift lander or offer weights toward arms with higher conversion rate.",
				ParametersSchema: []PresetParameter{
					{Key: "lookback_minutes", Type: "integer", Description: "Observation window minutes", Default: 1440, Min: ptrFloat(60), Max: ptrFloat(10080)},
					{Key: "min_clicks", Type: "integer", Description: "Minimum clicks per arm", Default: 100, Min: ptrFloat(100), Max: ptrFloat(100000)},
					{Key: "eval_interval_minutes", Type: "integer", Description: "Evaluation interval minutes", Default: 15, Min: ptrFloat(5), Max: ptrFloat(60)},
					{Key: "cooldown_minutes", Type: "integer", Description: "Cooldown after weight apply", Default: 60, Min: ptrFloat(15), Max: ptrFloat(10080)},
					{Key: "max_weight_delta_pct", Type: "integer", Description: "Max weight change per tick percent", Default: 50, Min: ptrFloat(10), Max: ptrFloat(100)},
				},
			},
			expand: func(params map[string]float64) (ExpandedRule, error) {
				return ExpandedRule{
					Name:                "CR best performer",
					Scope:               ScopeLander,
					Objective:           ObjectiveCR,
					Algorithm:           AlgorithmThompson,
					LookbackMinutes:     presetInt(params, "lookback_minutes", 1440),
					MinClicks:           presetInt(params, "min_clicks", 100),
					EvalIntervalMinutes: presetInt(params, "eval_interval_minutes", 15),
					CooldownMinutes:     presetInt(params, "cooldown_minutes", 60),
					MaxWeightDeltaPct:   presetInt(params, "max_weight_delta_pct", 50),
				}, nil
			},
		},
		{
			Preset: Preset{
				Key:         "epc_best_performer",
				Title:       "EPC best performer",
				Description: "Shift weights toward arms with higher earnings per click.",
				ParametersSchema: []PresetParameter{
					{Key: "lookback_minutes", Type: "integer", Description: "Observation window minutes", Default: 1440, Min: ptrFloat(60), Max: ptrFloat(10080)},
					{Key: "min_clicks", Type: "integer", Description: "Minimum clicks per arm", Default: 100, Min: ptrFloat(100), Max: ptrFloat(100000)},
					{Key: "eval_interval_minutes", Type: "integer", Description: "Evaluation interval minutes", Default: 15, Min: ptrFloat(5), Max: ptrFloat(60)},
					{Key: "cooldown_minutes", Type: "integer", Description: "Cooldown after weight apply", Default: 60, Min: ptrFloat(15), Max: ptrFloat(10080)},
					{Key: "max_weight_delta_pct", Type: "integer", Description: "Max weight change per tick percent", Default: 50, Min: ptrFloat(10), Max: ptrFloat(100)},
				},
			},
			expand: func(params map[string]float64) (ExpandedRule, error) {
				return ExpandedRule{
					Name:                "EPC best performer",
					Scope:               ScopeLander,
					Objective:           ObjectiveEPC,
					Algorithm:           AlgorithmProportional,
					LookbackMinutes:     presetInt(params, "lookback_minutes", 1440),
					MinClicks:           presetInt(params, "min_clicks", 100),
					EvalIntervalMinutes: presetInt(params, "eval_interval_minutes", 15),
					CooldownMinutes:     presetInt(params, "cooldown_minutes", 60),
					MaxWeightDeltaPct:   presetInt(params, "max_weight_delta_pct", 50),
				}, nil
			},
		},
		{
			Preset: Preset{
				Key:         "revenue_best_performer",
				Title:       "Revenue best performer",
				Description: "Shift weights toward arms with higher total revenue in the window.",
				ParametersSchema: []PresetParameter{
					{Key: "lookback_minutes", Type: "integer", Description: "Observation window minutes", Default: 1440, Min: ptrFloat(60), Max: ptrFloat(10080)},
					{Key: "min_clicks", Type: "integer", Description: "Minimum clicks per arm", Default: 100, Min: ptrFloat(100), Max: ptrFloat(100000)},
					{Key: "eval_interval_minutes", Type: "integer", Description: "Evaluation interval minutes", Default: 15, Min: ptrFloat(5), Max: ptrFloat(60)},
					{Key: "cooldown_minutes", Type: "integer", Description: "Cooldown after weight apply", Default: 60, Min: ptrFloat(15), Max: ptrFloat(10080)},
					{Key: "max_weight_delta_pct", Type: "integer", Description: "Max weight change per tick percent", Default: 50, Min: ptrFloat(10), Max: ptrFloat(100)},
				},
			},
			expand: func(params map[string]float64) (ExpandedRule, error) {
				return ExpandedRule{
					Name:                "Revenue best performer",
					Scope:               ScopeOffer,
					Objective:           ObjectiveRevenue,
					Algorithm:           AlgorithmProportional,
					LookbackMinutes:     presetInt(params, "lookback_minutes", 1440),
					MinClicks:           presetInt(params, "min_clicks", 100),
					EvalIntervalMinutes: presetInt(params, "eval_interval_minutes", 15),
					CooldownMinutes:     presetInt(params, "cooldown_minutes", 60),
					MaxWeightDeltaPct:   presetInt(params, "max_weight_delta_pct", 50),
				}, nil
			},
		},
		{
			Preset: Preset{
				Key:         "roi_best_performer",
				Title:       "ROI best performer",
				Description: "Shift weights toward arms with higher return on ad spend.",
				ParametersSchema: []PresetParameter{
					{Key: "lookback_minutes", Type: "integer", Description: "Observation window minutes", Default: 1440, Min: ptrFloat(60), Max: ptrFloat(10080)},
					{Key: "min_clicks", Type: "integer", Description: "Minimum clicks per arm", Default: 100, Min: ptrFloat(100), Max: ptrFloat(100000)},
					{Key: "min_spend_micro", Type: "integer", Description: "Minimum spend micro-units per arm", Default: 1_000_000, Min: ptrFloat(1)},
					{Key: "eval_interval_minutes", Type: "integer", Description: "Evaluation interval minutes", Default: 15, Min: ptrFloat(5), Max: ptrFloat(60)},
					{Key: "cooldown_minutes", Type: "integer", Description: "Cooldown after weight apply", Default: 60, Min: ptrFloat(15), Max: ptrFloat(10080)},
					{Key: "max_weight_delta_pct", Type: "integer", Description: "Max weight change per tick percent", Default: 50, Min: ptrFloat(10), Max: ptrFloat(100)},
				},
			},
			expand: func(params map[string]float64) (ExpandedRule, error) {
				return ExpandedRule{
					Name:                "ROI best performer",
					Scope:               ScopeLander,
					Objective:           ObjectiveROI,
					Algorithm:           AlgorithmProportional,
					LookbackMinutes:     presetInt(params, "lookback_minutes", 1440),
					MinClicks:           presetInt(params, "min_clicks", 100),
					MinSpendMicro:       int64(presetFloat(params, "min_spend_micro", 1_000_000)),
					EvalIntervalMinutes: presetInt(params, "eval_interval_minutes", 15),
					CooldownMinutes:     presetInt(params, "cooldown_minutes", 60),
					MaxWeightDeltaPct:   presetInt(params, "max_weight_delta_pct", 50),
				}, nil
			},
		},
	}
}

func ptrFloat(v float64) *float64 {
	return &v
}
