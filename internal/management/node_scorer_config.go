package management

import (
	"encoding/json"
	"fmt"
	"math"
)

const (
	scoringWeightsSettingKey = "scoring_weights_json"
	weightSumEpsilon         = 1e-6
)

type ScoringWeightsByRole map[string]map[string]float64

func ParseScoringWeightsJSON(raw string) (ScoringWeightsByRole, error) {
	raw = trimJSON(raw)
	if raw == "" {
		return nil, nil
	}
	var parsed ScoringWeightsByRole
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil, fmt.Errorf("parse scoring weights json: %w", err)
	}
	if err := ValidateScoringWeightsByRole(parsed); err != nil {
		return nil, err
	}
	return parsed, nil
}

func ValidateScoringWeightsByRole(byRole ScoringWeightsByRole) error {
	if len(byRole) == 0 {
		return nil
	}
	for role, weights := range byRole {
		if len(weights) == 0 {
			return fmt.Errorf("scoring weights role=%s: empty metric map", role)
		}
		if err := validateRoleMetricWeights(role, weights); err != nil {
			return err
		}
	}
	return nil
}

func validateRoleMetricWeights(role string, weights map[string]float64) error {
	defs := MetricsForRole(role)
	if len(defs) == 0 {
		return fmt.Errorf("scoring weights role=%s: unknown role", role)
	}
	known := make(map[string]struct{}, len(defs))
	for _, def := range defs {
		known[def.Name] = struct{}{}
	}
	var sum float64
	for name, w := range weights {
		if _, ok := known[name]; !ok {
			return fmt.Errorf("scoring weights role=%s: unknown metric %q", role, name)
		}
		if w < 0 || math.IsNaN(w) || math.IsInf(w, 0) {
			return fmt.Errorf("scoring weights role=%s metric=%s: invalid weight %v", role, name, w)
		}
		sum += w
	}
	for _, def := range defs {
		if _, ok := weights[def.Name]; !ok {
			return fmt.Errorf("scoring weights role=%s: missing metric %q", role, def.Name)
		}
	}
	if math.Abs(sum-1.0) > weightSumEpsilon {
		return fmt.Errorf("scoring weights role=%s: sum=%f want 1", role, sum)
	}
	return nil
}

func RenormalizeMetricWeights(weights map[string]float64) map[string]float64 {
	if len(weights) == 0 {
		return weights
	}
	var sum float64
	for _, w := range weights {
		sum += w
	}
	if sum <= 0 {
		return weights
	}
	out := make(map[string]float64, len(weights))
	for name, w := range weights {
		out[name] = w / sum
	}
	return out
}

func ApplyScoringWeights(defs []ScoringMetricDef, weights map[string]float64) []ScoringMetricDef {
	out := make([]ScoringMetricDef, len(defs))
	copy(out, defs)
	if len(weights) == 0 {
		return out
	}
	for i := range out {
		if w, ok := weights[out[i].Name]; ok {
			out[i].Weight = w
		}
	}
	return out
}

func BuildScoringMetricDefsByRole(byRole ScoringWeightsByRole) map[string][]ScoringMetricDef {
	roles := []string{RoleTracker, RoleRegionProxy, RoleProcessor}
	out := make(map[string][]ScoringMetricDef, len(roles))
	for _, role := range roles {
		defs := MetricsForRole(role)
		if weights, ok := byRole[role]; ok {
			defs = ApplyScoringWeights(defs, weights)
		}
		out[role] = defs
	}
	return out
}

func trimJSON(raw string) string {
	for len(raw) > 0 && (raw[0] == ' ' || raw[0] == '\n' || raw[0] == '\t' || raw[0] == '\r') {
		raw = raw[1:]
	}
	for len(raw) > 0 {
		last := raw[len(raw)-1]
		if last != ' ' && last != '\n' && last != '\t' && last != '\r' {
			break
		}
		raw = raw[:len(raw)-1]
	}
	return raw
}
