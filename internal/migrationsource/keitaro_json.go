package migrationsource

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

type keitaroExport struct {
	Campaigns []keitaroCampaign `json:"campaigns"`
}

type keitaroCampaign struct {
	ID            int     `json:"id"`
	Name          string  `json:"name"`
	TrafficSource string  `json:"traffic_source"`
	Budget        float64 `json:"budget"`
	TrackingURL   string  `json:"tracking_url"`
	LanderURL     string  `json:"lander_url"`
	PostbackURL   string  `json:"postback_url"`
}

func ParseKeitaroJSON(payload []byte) (NormalizedBundle, error) {
	payload = bytesTrimSpace(payload)
	if len(payload) == 0 {
		return NormalizedBundle{}, fmt.Errorf("empty payload")
	}
	var campaigns []keitaroCampaign
	if payload[0] == '[' {
		if err := json.Unmarshal(payload, &campaigns); err != nil {
			return NormalizedBundle{}, fmt.Errorf("decode keitaro campaign array: %w", err)
		}
	} else {
		var doc keitaroExport
		if err := json.Unmarshal(payload, &doc); err != nil {
			return NormalizedBundle{}, fmt.Errorf("decode keitaro export: %w", err)
		}
		campaigns = doc.Campaigns
	}
	if len(campaigns) == 0 {
		return NormalizedBundle{}, fmt.Errorf("no campaigns in payload")
	}
	out := NormalizedBundle{SourceKind: SourceKindKeitaroJSON}
	for i, row := range campaigns {
		name := strings.TrimSpace(row.Name)
		if name == "" {
			return NormalizedBundle{}, fmt.Errorf("campaign index %d missing name", i)
		}
		ref := fmt.Sprintf("keitaro:%d", row.ID)
		if row.ID == 0 {
			ref = fmt.Sprintf("keitaro:%d", i+1)
		}
		out.Campaigns = append(out.Campaigns, NormalizedCampaign{
			Ref:               ref,
			Name:              name,
			TrafficSourceName: strings.TrimSpace(row.TrafficSource),
			TrackingURL:       strings.TrimSpace(row.TrackingURL),
			LanderURL:         strings.TrimSpace(row.LanderURL),
			PostbackURL:       strings.TrimSpace(row.PostbackURL),
			BudgetUSD:         row.Budget,
		})
	}
	return out, nil
}

func parseClickQueryParams(trackingURL string) (map[string]string, error) {
	trackingURL = strings.TrimSpace(trackingURL)
	if trackingURL == "" {
		return nil, nil
	}
	u, err := url.Parse(trackingURL)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	if len(q) == 0 {
		return nil, nil
	}
	out := make(map[string]string, len(q))
	for key, vals := range q {
		if len(vals) == 0 {
			continue
		}
		out[key] = vals[0]
	}
	return out, nil
}

func budgetUSDToMicro(usd float64) int64 {
	if usd <= 0 {
		return 0
	}
	return int64(usd * 1_000_000)
}

func bytesTrimSpace(b []byte) []byte {
	return []byte(strings.TrimSpace(string(b)))
}
