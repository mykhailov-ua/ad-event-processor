package campaign

import (
	"fmt"
	"strings"

	"ad-event-processor/internal/integrationschema"
)

const (
	maxClickQueryParamKeys     = 40
	maxClickQueryParamValueLen = 512
	maxTrafficTemplateIDLen    = 64
)

var allowedClickQueryKeys = func() map[string]bool {
	keys := map[string]bool{
		"ad_campaign_id": true,
		"fbclid":         true,
		"gclid":          true,
		"ttclid":         true,
	}
	for i := 1; i <= 30; i++ {
		keys[fmt.Sprintf("sub%d", i)] = true
	}
	return keys
}()

func validateTrafficTemplateID(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil
	}
	if len(id) > maxTrafficTemplateIDLen {
		return fmt.Errorf("traffic_template_id too long")
	}
	return nil
}

func validateClickQueryParams(params map[string]string) error {
	if len(params) > maxClickQueryParamKeys {
		return fmt.Errorf("click_query_params: too many keys")
	}
	for key, value := range params {
		if !allowedClickQueryKeys[key] {
			return fmt.Errorf("click_query_params: invalid key %q", key)
		}
		if len(value) > maxClickQueryParamValueLen {
			return fmt.Errorf("click_query_params: value too long for %q", key)
		}
	}
	return nil
}

func normalizeClickQueryParams(params map[string]string) map[string]string {
	if len(params) == 0 {
		return nil
	}
	out := make(map[string]string, len(params))
	for key, value := range params {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		out[key] = value
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func validateWizardTrafficSourceStep(step CampaignWizardTrafficSourceStep) error {
	if strings.TrimSpace(step.Name) == "" {
		return errValidation("name is required")
	}
	if err := validateTrafficTemplateID(step.TrafficTemplateID); err != nil {
		return errValidation(err.Error())
	}
	if step.ClickQueryParams != nil {
		if err := validateClickQueryParams(step.ClickQueryParams); err != nil {
			return errValidation(err.Error())
		}
	}
	return nil
}

func validateWizardIntegrationTemplateStep(step CampaignWizardIntegrationTemplateStep) error {
	name := strings.TrimSpace(step.IntegrationSchema)
	if name == "" {
		return errValidation("integration_schema is required")
	}
	if _, ok := integrationschema.FindCatalogEntry(name); !ok {
		return errValidation(fmt.Sprintf("integration schema %q not found in catalog", name))
	}
	if net := strings.TrimSpace(step.AffiliateNetwork); net != "" {
		if _, ok := integrationschema.FindCatalogEntry(net); !ok {
			return errValidation(fmt.Sprintf("affiliate network schema %q not found in catalog", net))
		}
	}
	return nil
}

func validateWizardFlowSkeletonStep(step CampaignWizardFlowSkeletonStep) error {
	if strings.TrimSpace(step.FlowName) == "" {
		return errValidation("flow_name is required")
	}
	if strings.TrimSpace(step.Lander.Name) == "" {
		return errValidation("lander.name is required")
	}
	if strings.TrimSpace(step.Lander.URL) == "" {
		return errValidation("lander.url is required")
	}
	if strings.TrimSpace(step.Offer.Name) == "" || strings.TrimSpace(step.Offer.URL) == "" {
		return errValidation("offer.name and offer.url are required")
	}
	return nil
}

func validateWizardBudgetStep(step CampaignWizardBudgetStep) error {
	if step.BudgetLimitMicro <= 0 {
		return errValidation("budget_limit_micro must be positive")
	}
	return nil
}
