package costsync

import (
	"encoding/json"
	"strings"
)

const (
	AttributionModeToken  = "token"
	AttributionModeSpread = "spread"
)

type TokenMapping struct {
	PlacementField  string `json:"placement_field"`
	NetworkObject   string `json:"network_object"`
	AttributionMode string `json:"attribution_mode"`
}

func ParseTokenMapping(raw []byte) TokenMapping {
	out := TokenMapping{
		PlacementField:  "placement_id",
		NetworkObject:   "ad_id",
		AttributionMode: AttributionModeToken,
	}
	if len(raw) == 0 {
		return out
	}
	var parsed TokenMapping
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return out
	}
	if field := strings.TrimSpace(parsed.PlacementField); field != "" {
		out.PlacementField = field
	}
	if obj := strings.TrimSpace(parsed.NetworkObject); obj != "" {
		out.NetworkObject = obj
	}
	switch strings.ToLower(strings.TrimSpace(parsed.AttributionMode)) {
	case AttributionModeSpread:
		out.AttributionMode = AttributionModeSpread
	default:
		out.AttributionMode = AttributionModeToken
	}
	return out
}

func ValidSyncIntervalMinutes(interval int) bool {
	switch interval {
	case 15, 30, 60, 1440:
		return true
	default:
		return false
	}
}
