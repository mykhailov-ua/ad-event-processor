package domain

import "strings"

type TimezoneAttestationMode string

const (
	TimezoneAttestationModeOff            TimezoneAttestationMode = "off"
	TimezoneAttestationModeIPCountry      TimezoneAttestationMode = "ip_country"
	TimezoneAttestationModeCampaignTarget TimezoneAttestationMode = "campaign_target"
	TimezoneAttestationModeStrict         TimezoneAttestationMode = "strict"
)

func ParseTimezoneAttestationMode(s string) TimezoneAttestationMode {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case string(TimezoneAttestationModeOff):
		return TimezoneAttestationModeOff
	case string(TimezoneAttestationModeIPCountry):
		return TimezoneAttestationModeIPCountry
	case string(TimezoneAttestationModeCampaignTarget):
		return TimezoneAttestationModeCampaignTarget
	case string(TimezoneAttestationModeStrict):
		return TimezoneAttestationModeStrict
	default:
		return ""
	}
}

// Effective maps unset DB values to ip_country so legacy campaigns keep prior safe-page timezone behavior.
func (m TimezoneAttestationMode) Effective() TimezoneAttestationMode {
	if m == "" {
		return TimezoneAttestationModeIPCountry
	}
	return m
}

func applyCampaignTimezoneAttestation(camp *Campaign, raw string) {
	if camp == nil {
		return
	}
	camp.TimezoneAttestationMode = ParseTimezoneAttestationMode(raw)
}
