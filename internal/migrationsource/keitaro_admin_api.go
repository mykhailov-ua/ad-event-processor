package migrationsource

import (
	"encoding/json"
	"fmt"
	"strings"
)

type keitaroAdminCampaign struct {
	ID              int             `json:"id"`
	Name            string          `json:"name"`
	Alias           string          `json:"alias"`
	State           string          `json:"state"`
	CostType        string          `json:"cost_type"`
	CostValue       json.RawMessage `json:"cost_value"`
	CostCurrency    string          `json:"cost_currency"`
	Parameters      string          `json:"parameters"`
	TrafficSourceID int             `json:"traffic_source_id"`
	Domain          string          `json:"domain"`
	TrackerDomain   string          `json:"tracker_domain"`
	TrafficSource   string          `json:"traffic_source"`
	LanderURL       string          `json:"lander_url"`
	PostbackURL     string          `json:"postback_url"`
	Budget          float64         `json:"budget"`
	TrackingURL     string          `json:"tracking_url"`
}

type keitaroAdminExport struct {
	Campaigns []keitaroAdminCampaign `json:"campaigns"`
}

func ParseKeitaroAdminAPI(payload []byte) (NormalizedBundle, error) {
	var campaigns []keitaroAdminCampaign
	err := decodeJSONArrayOrCampaignsWrapper(payload,
		func(b []byte) error {
			return json.Unmarshal(b, &campaigns)
		},
		func(b []byte) error {
			var doc keitaroAdminExport
			if err := json.Unmarshal(b, &doc); err != nil {
				return fmt.Errorf("decode keitaro admin api export: %w", err)
			}
			campaigns = doc.Campaigns
			return nil
		},
	)
	if err != nil {
		return NormalizedBundle{}, err
	}
	if len(campaigns) == 0 {
		return NormalizedBundle{}, fmt.Errorf("no campaigns in payload")
	}
	out := NormalizedBundle{SourceKind: SourceKindKeitaroAdminAPI}
	for i, row := range campaigns {
		name := strings.TrimSpace(row.Name)
		if name == "" {
			return NormalizedBundle{}, fmt.Errorf("campaign index %d missing name", i)
		}
		if strings.TrimSpace(row.Alias) == "" && strings.TrimSpace(row.TrackingURL) == "" {
			return NormalizedBundle{}, fmt.Errorf("campaign index %d missing alias (Admin API wire)", i)
		}
		domain := strings.TrimSpace(row.Domain)
		if domain == "" {
			domain = strings.TrimSpace(row.TrackerDomain)
		}
		trackingURL, err := keitaroAdminTrackingURL(domain, row.Alias, row.Parameters, row.TrackingURL)
		if err != nil {
			return NormalizedBundle{}, fmt.Errorf("campaign index %d: %w", i, err)
		}
		ref := fmt.Sprintf("keitaro:%d", row.ID)
		if row.ID == 0 {
			ref = fmt.Sprintf("keitaro:%d", i+1)
		}
		out.Campaigns = append(out.Campaigns, NormalizedCampaign{
			Ref:               ref,
			Name:              name,
			TrafficSourceName: strings.TrimSpace(row.TrafficSource),
			TrackingURL:       trackingURL,
			LanderURL:         strings.TrimSpace(row.LanderURL),
			PostbackURL:       strings.TrimSpace(row.PostbackURL),
			BudgetUSD:         row.Budget,
		})
	}
	return out, nil
}
