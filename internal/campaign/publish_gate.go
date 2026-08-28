package campaign

import (
	"context"
	"fmt"
	"strings"

	"ad-event-processor/pkg/campaignmacro"
	"ad-event-processor/pkg/proxyupstream"

	"github.com/google/uuid"
)

type PublishGateEvalInput struct {
	CampaignID          uuid.UUID
	Name                string
	BudgetLimit         int64
	CurrentSpend        int64
	TargetCountries     []string
	TargetURL           string
	ClickQueryParams    map[string]string
	ClickDelivery       string
	ProxyUpstreamURL    string
	AllowHTTPInsecure   bool
	FlowMissing         bool
	FlowPathError       string
	IntegrationHealth   IntegrationHealthDTO
	IntegrationHealthOK bool
}

func EvaluatePublishBlocked(input PublishGateEvalInput) *CampaignPublishBlockedError {
	fieldErrors := make(map[string]string)
	var warningSlugs []string

	if strings.TrimSpace(input.Name) == "" {
		fieldErrors["name"] = "name is required"
	}
	if input.BudgetLimit <= 0 {
		fieldErrors["budget_limit"] = "budget must be positive"
	}
	if input.BudgetLimit < input.CurrentSpend {
		fieldErrors["budget_limit"] = "budget_limit cannot be below current spend"
	}
	if len(input.TargetCountries) == 0 {
		warningSlugs = append(warningSlugs, "target_countries_empty")
	}

	if input.FlowMissing {
		fieldErrors["flow_id"] = "campaign flow is required"
	} else if input.FlowPathError != "" {
		fieldErrors["flow_paths"] = input.FlowPathError
	}

	targetURL := strings.TrimSpace(input.TargetURL)
	if targetURL == "" {
		warningSlugs = append(warningSlugs, "target_url_empty")
	} else {
		macroCtx := campaignmacro.PreviewContext(input.CampaignID.String(), campaignmacro.PreviewRequest{
			Sub1:    "preview",
			Country: "US",
		})
		_, unresolved := campaignmacro.Expand(targetURL, macroCtx)
		if len(unresolved) > 0 {
			fieldErrors["target_url"] = fmt.Sprintf("unresolved macros: %s", strings.Join(unresolved, ", "))
		}
		if params := input.ClickQueryParams; len(params) > 0 {
			for key, value := range params {
				_, paramUnresolved := campaignmacro.Expand(value, macroCtx)
				if len(paramUnresolved) > 0 {
					fieldErrors["click_query_params."+key] = fmt.Sprintf("unresolved macros: %s", strings.Join(paramUnresolved, ", "))
				}
			}
		}
		if strings.HasPrefix(strings.ToLower(targetURL), "http://") {
			warningSlugs = append(warningSlugs, "macro_click_url_uses_http")
		}
	}

	if err := proxyupstream.ValidateDeliveryPair(context.Background(), input.ClickDelivery, input.ProxyUpstreamURL, input.AllowHTTPInsecure); err != nil {
		fieldErrors["click_delivery"] = err.Error()
	}

	if input.IntegrationHealthOK {
		for _, healthRow := range input.IntegrationHealth.Rows {
			if healthRow.Status == string(IntegrationHealthFail) {
				key := "integration_" + strings.TrimSpace(healthRow.Slug)
				if key == "integration_" {
					key = "integration"
				}
				fieldErrors[key] = healthRow.Message
			}
		}
	}

	if len(fieldErrors) == 0 {
		return nil
	}
	return &CampaignPublishBlockedError{
		FieldErrors:  fieldErrors,
		WarningSlugs: warningSlugs,
	}
}
