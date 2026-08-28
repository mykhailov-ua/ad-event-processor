package campaign

import (
	"fmt"
	"strings"
)

const campaignGeoExpandMaxRows = 50

type CampaignGeoCountryRowDTO struct {
	Code  string `json:"code"`
	Label string `json:"label"`
	Kind  string `json:"kind"`
}

type CampaignGeoSummaryDTO struct {
	IncludedLabel   string                     `json:"included_label"`
	ExcludedLabel   string                     `json:"excluded_label"`
	ConflictWarning bool                       `json:"conflict_warning,omitempty"`
	Expanded        []CampaignGeoCountryRowDTO `json:"expanded,omitempty"`
	Truncated       bool                       `json:"truncated,omitempty"`
}

var isoCountryLabels = map[string]string{
	"US": "United States",
	"GB": "United Kingdom",
	"DE": "Germany",
	"FR": "France",
	"CA": "Canada",
	"AU": "Australia",
}

func buildCampaignGeoSummary(campaign CampaignDTO, expand bool) CampaignGeoSummaryDTO {
	include := normalizeCountryCodes(campaign.TargetCountries)
	out := CampaignGeoSummaryDTO{
		IncludedLabel: geoListLabel(include, "any country"),
		ExcludedLabel: "none",
	}
	if !expand {
		return out
	}
	rows := make([]CampaignGeoCountryRowDTO, 0, len(include))
	for _, code := range include {
		rows = append(rows, CampaignGeoCountryRowDTO{Code: code, Label: countryLabel(code), Kind: "include"})
	}
	if len(rows) > campaignGeoExpandMaxRows {
		out.Truncated = true
		rows = rows[:campaignGeoExpandMaxRows]
	}
	out.Expanded = rows
	return out
}

func normalizeCountryCodes(codes []string) []string {
	out := make([]string, 0, len(codes))
	seen := make(map[string]struct{}, len(codes))
	for _, raw := range codes {
		code := strings.ToUpper(strings.TrimSpace(raw))
		if code == "" {
			continue
		}
		if _, dup := seen[code]; dup {
			continue
		}
		seen[code] = struct{}{}
		out = append(out, code)
	}
	return out
}

func geoListLabel(codes []string, emptyLabel string) string {
	if len(codes) == 0 {
		return emptyLabel
	}
	if len(codes) == 1 {
		return countryLabel(codes[0])
	}
	return fmt.Sprintf("%d countries", len(codes))
}

func countryLabel(code string) string {
	if label, ok := isoCountryLabels[code]; ok {
		return label
	}
	return code
}
