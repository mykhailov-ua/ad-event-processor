package migrationsource

import (
	"encoding/json"
	"fmt"
	"strings"
)

type binomExport struct {
	Campaigns []binomCampaign `json:"campaigns"`
}

type binomCampaign struct {
	ID                int     `json:"id"`
	Name              string  `json:"name"`
	TrafficSource     string  `json:"traffic_source"`
	TrafficSourceName string  `json:"traffic_source_name"`
	Budget            float64 `json:"budget"`
	TrackingURL       string  `json:"tracking_url"`
	LanderURL         string  `json:"lander_url"`
	PostbackURL       string  `json:"postback_url"`
}

func ParseBinomJSON(payload []byte) (NormalizedBundle, error) {
	payload = bytesTrimSpace(payload)
	if len(payload) == 0 {
		return NormalizedBundle{}, fmt.Errorf("empty payload")
	}
	var campaigns []binomCampaign
	if payload[0] == '[' {
		if err := json.Unmarshal(payload, &campaigns); err != nil {
			return NormalizedBundle{}, fmt.Errorf("decode binom campaign array: %w", err)
		}
	} else {
		var doc binomExport
		if err := json.Unmarshal(payload, &doc); err != nil {
			return NormalizedBundle{}, fmt.Errorf("decode binom export: %w", err)
		}
		campaigns = doc.Campaigns
	}
	if len(campaigns) == 0 {
		return NormalizedBundle{}, fmt.Errorf("no campaigns in payload")
	}
	out := NormalizedBundle{SourceKind: SourceKindBinomJSON}
	for i, row := range campaigns {
		name := strings.TrimSpace(row.Name)
		if name == "" {
			return NormalizedBundle{}, fmt.Errorf("campaign index %d missing name", i)
		}
		ref := fmt.Sprintf("binom:%d", row.ID)
		if row.ID == 0 {
			ref = fmt.Sprintf("binom:%d", i+1)
		}
		sourceName := strings.TrimSpace(row.TrafficSourceName)
		if sourceName == "" {
			sourceName = strings.TrimSpace(row.TrafficSource)
		}
		trackingURL := strings.TrimSpace(row.TrackingURL)
		if trackingURL == "" {
			return NormalizedBundle{}, fmt.Errorf("campaign index %d missing tracking_url (use source_kind binom_report_api for report API wire)", i)
		}
		out.Campaigns = append(out.Campaigns, NormalizedCampaign{
			Ref:               ref,
			Name:              name,
			TrafficSourceName: sourceName,
			TrackingURL:       trackingURL,
			LanderURL:         strings.TrimSpace(row.LanderURL),
			PostbackURL:       strings.TrimSpace(row.PostbackURL),
			BudgetUSD:         row.Budget,
		})
	}
	return out, nil
}
