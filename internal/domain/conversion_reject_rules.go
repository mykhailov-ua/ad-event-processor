package domain

import "encoding/json"

// ConversionRejectRules overrides processor settlement smart-reject per campaign.
// Null fields inherit deployment env defaults (CONVERSION_REJECT_*).
type ConversionRejectRules struct {
	Enabled            *bool `json:"enabled,omitempty"`
	MinTTCDurationMs   *int  `json:"min_ttc_ms,omitempty"`
	RejectNoClick      *bool `json:"reject_no_click,omitempty"`
	RejectLowTTC       *bool `json:"reject_low_ttc,omitempty"`
	RejectDuplicate    *bool `json:"reject_duplicate,omitempty"`
	RejectIPDrift      *bool `json:"reject_ip_drift,omitempty"`
	RejectDatacenterIP *bool `json:"reject_datacenter_ip,omitempty"`
}

func ParseConversionRejectRulesJSON(raw []byte) ConversionRejectRules {
	if len(raw) == 0 {
		return ConversionRejectRules{}
	}
	var rules ConversionRejectRules
	if err := json.Unmarshal(raw, &rules); err != nil {
		return ConversionRejectRules{}
	}
	return rules
}

func MarshalConversionRejectRules(rules ConversionRejectRules) ([]byte, error) {
	return json.Marshal(rules)
}

func MergeConversionRejectRulesPatch(base, patch ConversionRejectRules) ConversionRejectRules {
	out := base
	if patch.Enabled != nil {
		out.Enabled = patch.Enabled
	}
	if patch.MinTTCDurationMs != nil {
		out.MinTTCDurationMs = patch.MinTTCDurationMs
	}
	if patch.RejectNoClick != nil {
		out.RejectNoClick = patch.RejectNoClick
	}
	if patch.RejectLowTTC != nil {
		out.RejectLowTTC = patch.RejectLowTTC
	}
	if patch.RejectDuplicate != nil {
		out.RejectDuplicate = patch.RejectDuplicate
	}
	if patch.RejectIPDrift != nil {
		out.RejectIPDrift = patch.RejectIPDrift
	}
	if patch.RejectDatacenterIP != nil {
		out.RejectDatacenterIP = patch.RejectDatacenterIP
	}
	return out
}
