package campaign

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type IntegrationHealthInput struct {
	CampaignID                uuid.UUID
	IntegrationSchemaBound    bool
	TrafficTemplateID         string
	TargetURL                 string
	ClickQueryParams          map[string]string
	IngressCostConfigured     bool
	IngressCostParam          string
	PostbackConfigured        bool
	CostSyncNetwork           string
	CostSyncCredentialPresent bool
}

var trafficTemplateCostSyncNetwork = map[string]string{
	"meta-facebook":  "facebook",
	"meta-instagram": "facebook",
	"google-ads":     "google",
	"google-display": "google",
	"youtube-ads":    "google",
	"tiktok-ads":     "tiktok",
}

var costSyncRequiredKeys = map[string][]string{
	"facebook": {"ad_campaign_id", "sub2", "sub4", "fbclid"},
	"google":   {"ad_campaign_id", "sub2", "sub3", "gclid"},
	"tiktok":   {"ad_campaign_id", "sub2", "sub3", "ttclid"},
}

func BuildCampaignIntegrationHealth(input IntegrationHealthInput) IntegrationHealthDTO {
	rows := make([]IntegrationHealthRow, 0, 6)
	trackingRoute := fmt.Sprintf("/campaigns/%s?tab=tracking", input.CampaignID)
	configRoute := fmt.Sprintf("/campaigns/%s?tab=config", input.CampaignID)
	postbackRoute := fmt.Sprintf("/campaigns/%s?tab=postbacks", input.CampaignID)
	costSyncRoute := "/integrations/cost-sync"

	if input.IntegrationSchemaBound {
		rows = append(rows, IntegrationHealthRow{
			Slug:    "integration_schema",
			Status:  string(IntegrationHealthOK),
			Message: "Integration schema is bound on the campaign.",
		})
	} else if template := strings.TrimSpace(input.TrafficTemplateID); template != "" && template != "direct-custom" {
		rows = append(rows, IntegrationHealthRow{
			Slug:    "traffic_template",
			Status:  string(IntegrationHealthOK),
			Message: "Traffic template preset is saved on the campaign.",
		})
	} else {
		rows = append(rows, IntegrationHealthRow{
			Slug:     "integration_schema",
			Status:   string(IntegrationHealthWarn),
			Message:  "No integration schema or traffic template preset; apply a bundled template on Integration.",
			FixRoute: trackingRoute,
		})
	}

	if input.TargetURL == "" {
		rows = append(rows, IntegrationHealthRow{
			Slug:     "target_url",
			Status:   string(IntegrationHealthWarn),
			Message:  "Target URL is empty; clicks have nowhere to land after tracking.",
			FixRoute: configRoute,
		})
	} else {
		rows = append(rows, IntegrationHealthRow{
			Slug:    "target_url",
			Status:  string(IntegrationHealthOK),
			Message: "Target URL is configured.",
		})
	}

	if network := input.CostSyncNetwork; network != "" {
		required := costSyncRequiredKeys[network]
		missing := missingClickJoinKeys(input.ClickQueryParams, required)
		if len(missing) > 0 {
			rows = append(rows, IntegrationHealthRow{
				Slug:     "click_join_keys",
				Status:   string(IntegrationHealthFail),
				Message:  fmt.Sprintf("Click preset missing Cost Sync join keys: %s", strings.Join(missing, ", ")),
				FixRoute: trackingRoute,
			})
		} else {
			rows = append(rows, IntegrationHealthRow{
				Slug:    "click_join_keys",
				Status:  string(IntegrationHealthOK),
				Message: "Required Cost Sync join keys are present in the click preset.",
			})
		}
		if !input.CostSyncCredentialPresent {
			rows = append(rows, IntegrationHealthRow{
				Slug:     "cost_sync_credential",
				Status:   string(IntegrationHealthWarn),
				Message:  fmt.Sprintf("No Cost Sync credential for network %s; spend join stays empty.", network),
				FixRoute: costSyncRoute,
			})
		} else {
			rows = append(rows, IntegrationHealthRow{
				Slug:    "cost_sync_credential",
				Status:  string(IntegrationHealthOK),
				Message: fmt.Sprintf("Cost Sync credential configured for %s.", network),
			})
		}
	}

	if input.PostbackConfigured {
		rows = append(rows, IntegrationHealthRow{
			Slug:    "postback_config",
			Status:  string(IntegrationHealthOK),
			Message: "Postback or CAPI template is configured.",
		})
	} else {
		rows = append(rows, IntegrationHealthRow{
			Slug:     "postback_config",
			Status:   string(IntegrationHealthWarn),
			Message:  "No postback config; affiliate conversions will not forward until CAPI & Postbacks is set.",
			FixRoute: postbackRoute,
		})
	}

	if ingressMacroInPreset(input.ClickQueryParams) {
		if input.IngressCostConfigured {
			rows = append(rows, IntegrationHealthRow{
				Slug:    "ingress_cost_config",
				Status:  string(IntegrationHealthOK),
				Message: fmt.Sprintf("Ingress cost parsing enabled for param %s.", input.IngressCostParam),
			})
		} else {
			rows = append(rows, IntegrationHealthRow{
				Slug:     "ingress_cost_config",
				Status:   string(IntegrationHealthWarn),
				Message:  "Click preset includes an ingress cost macro but ingress_cost_config is unset.",
				FixRoute: trackingRoute,
			})
		}
	}

	return IntegrationHealthDTO{
		CampaignID: input.CampaignID.String(),
		Summary:    summarizeIntegrationHealth(rows),
		Rows:       rows,
	}
}

func missingClickJoinKeys(params map[string]string, required []string) []string {
	if len(required) == 0 {
		return nil
	}
	var missing []string
	for _, key := range required {
		if strings.TrimSpace(params[key]) == "" {
			missing = append(missing, key)
		}
	}
	return missing
}

func ingressMacroInPreset(params map[string]string) bool {
	for _, key := range []string{"cost", "cpc", "bid"} {
		val := strings.TrimSpace(params[key])
		if val == "" {
			continue
		}
		if strings.Contains(val, "{cost}") || strings.Contains(val, "{cpc}") || strings.Contains(val, "{bid}") {
			return true
		}
	}
	return false
}

func summarizeIntegrationHealth(rows []IntegrationHealthRow) string {
	summary := string(IntegrationHealthOK)
	for _, row := range rows {
		switch IntegrationHealthStatus(row.Status) {
		case IntegrationHealthFail:
			return string(IntegrationHealthFail)
		case IntegrationHealthWarn:
			summary = string(IntegrationHealthWarn)
		}
	}
	return summary
}

func CostSyncNetworkForTrafficTemplate(templateID string) string {
	return trafficTemplateCostSyncNetwork[strings.TrimSpace(templateID)]
}
