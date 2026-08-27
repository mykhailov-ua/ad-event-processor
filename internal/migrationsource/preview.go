package migrationsource

import (
	"fmt"
	"strings"
)

func Preview(kind SourceKind, payload []byte, maps *Maps) (PreviewResult, error) {
	if len(payload) > MaxPayloadBytes {
		return PreviewResult{}, fmt.Errorf("payload exceeds %d bytes", MaxPayloadBytes)
	}
	if maps == nil {
		loaded, err := LoadMaps(MapsRootDir())
		if err != nil {
			return PreviewResult{}, err
		}
		maps = loaded
	}
	bundle, err := Parse(kind, payload)
	if err != nil {
		return PreviewResult{}, err
	}
	macroMapper := NewMacroMapper(maps.KeitaroMacros)
	if usesBinomMacroMaps(kind) {
		macroMapper = NewMacroMapper(maps.BinomMacros)
	}
	resolver := NewSchemaResolver(maps)

	out := PreviewResult{
		SourceKind:      kind,
		SecretsStripped: 0,
	}
	for _, camp := range bundle.Campaigns {
		mapped := MappedCampaign{
			Ref:                 camp.Ref,
			Name:                camp.Name,
			TrafficSourceName:   camp.TrafficSourceName,
			TargetURL:           camp.LanderURL,
			PostbackURLTemplate: camp.PostbackURL,
			BudgetLimitMicro:    budgetUSDToMicro(camp.BudgetUSD),
		}
		if camp.TrafficSourceName != "" {
			var row SourceEntry
			var ok bool
			if usesBinomMacroMaps(kind) {
				row, ok = resolver.ResolveBinom(camp.TrafficSourceName)
			} else {
				row, ok = resolver.ResolveKeitaro(camp.TrafficSourceName)
			}
			if ok {
				mapped.BundledSlug = strings.TrimSpace(row.BundledSlug)
				mapped.UITemplateID = strings.TrimSpace(row.UITemplateID)
				mapped.IntegrationSchemaName = mapped.BundledSlug
			} else {
				out.Warnings = append(out.Warnings, Warning{
					Slug:        "unknown_traffic_source",
					Message:     "no bundled slug for traffic source " + camp.TrafficSourceName,
					CampaignRef: camp.Ref,
				})
			}
		}
		if camp.LanderURL == "" && camp.TrackingURL != "" {
			out.Warnings = append(out.Warnings, Warning{
				Slug:        "lander_external_only",
				Message:     "lander URL empty; set target URL manually after import",
				CampaignRef: camp.Ref,
			})
		}
		rawParams, err := parseClickQueryParams(camp.TrackingURL)
		if err != nil {
			out.Warnings = append(out.Warnings, Warning{
				Slug:        "invalid_tracking_url",
				Message:     err.Error(),
				CampaignRef: camp.Ref,
			})
		} else if len(rawParams) > 0 {
			params, macroWarnings, ingress := macroMapper.ApplyQueryParams(rawParams)
			for _, w := range macroWarnings {
				w.CampaignRef = camp.Ref
				out.Warnings = append(out.Warnings, w)
			}
			mapped.ClickQueryParams = params
			mapped.IngressCostParam = ingress
		}
		if camp.PostbackURL != "" {
			out.SecretsStripped++
		}
		out.MappedCampaigns = append(out.MappedCampaigns, mapped)
	}
	return out, nil
}

func usesBinomMacroMaps(kind SourceKind) bool {
	switch kind {
	case SourceKindBinomJSON, SourceKindBinomReportAPI:
		return true
	default:
		return false
	}
}

func ListSources() []SourceKindMeta {
	return []SourceKindMeta{
		{Kind: SourceKindKeitaroJSON, Label: "Keitaro interchange JSON (normalized)"},
		{Kind: SourceKindKeitaroAdminAPI, Label: "Keitaro Admin API campaigns wire"},
		{Kind: SourceKindBinomJSON, Label: "Binom interchange JSON (normalized)"},
		{Kind: SourceKindBinomReportAPI, Label: "Binom campaign report API wire"},
		{Kind: SourceKindNativeV1, Label: "Native campaign export v1"},
	}
}

type SourceKindMeta struct {
	Kind  SourceKind `json:"kind"`
	Label string     `json:"label"`
}

type SourcesResponse struct {
	Sources         []SourceKindMeta `json:"sources"`
	MaxPayloadBytes int              `json:"max_payload_bytes"`
}
