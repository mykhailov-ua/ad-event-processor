package migrationsource

import (
	"fmt"
	"strings"
)

// Preview parses a foreign payload and applies macro/source maps without database writes.
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
	if kind == SourceKindBinomJSON {
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
			if row, ok := resolver.ResolveKeitaro(camp.TrafficSourceName); ok {
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

// ListSources returns supported migration source kinds for GET /migrate/sources.
func ListSources() []SourceKindMeta {
	return []SourceKindMeta{
		{Kind: SourceKindKeitaroJSON, Label: "Keitaro JSON export"},
		{Kind: SourceKindBinomJSON, Label: "Binom JSON export (planned)"},
		{Kind: SourceKindNativeV1, Label: "Native campaign export v1"},
	}
}

// SourceKindMeta describes a supported migration source.
type SourceKindMeta struct {
	Kind  SourceKind `json:"kind"`
	Label string     `json:"label"`
}

// SourcesResponse is returned by GET /api/v1/campaigns/migrate/sources.
type SourcesResponse struct {
	Sources         []SourceKindMeta `json:"sources"`
	MaxPayloadBytes int              `json:"max_payload_bytes"`
}
