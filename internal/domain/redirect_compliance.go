package domain

import "strings"

type RedirectComplianceMode string

const (
	RedirectComplianceStrict    RedirectComplianceMode = "strict"
	RedirectComplianceLegacyDMR RedirectComplianceMode = "legacy_dmr"
)

func ParseRedirectComplianceMode(raw string) RedirectComplianceMode {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case string(RedirectComplianceLegacyDMR), "legacy":
		return RedirectComplianceLegacyDMR
	default:
		return RedirectComplianceStrict
	}
}

func (m RedirectComplianceMode) AllowsDMR() bool {
	return m == RedirectComplianceLegacyDMR
}

func (c *Campaign) RedirectAllowsDMR() bool {
	if c == nil {
		return false
	}
	return c.RedirectComplianceMode.AllowsDMR()
}

func applyCampaignRedirectCompliance(camp *Campaign, raw string) {
	if camp == nil {
		return
	}
	camp.RedirectComplianceMode = ParseRedirectComplianceMode(raw)
}
