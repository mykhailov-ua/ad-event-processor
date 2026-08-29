package campaign

import (
	"fmt"
	"strings"

	"ad-event-processor/internal/integrationschema"
)

func validateWizardTrafficSourceStep(step CampaignWizardTrafficSourceStep) error {
	if strings.TrimSpace(step.Name) == "" {
		return ErrValidationf("name is required")
	}
	if err := ValidateTrafficTemplateID(step.TrafficTemplateID); err != nil {
		return ErrValidationf(err.Error())
	}
	if step.ClickQueryParams != nil {
		if err := ValidateClickQueryParams(step.ClickQueryParams); err != nil {
			return ErrValidationf(err.Error())
		}
	}
	return nil
}

func validateWizardIntegrationTemplateStep(step CampaignWizardIntegrationTemplateStep) error {
	name := strings.TrimSpace(step.IntegrationSchema)
	if name == "" {
		return ErrValidationf("integration_schema is required")
	}
	if _, ok := integrationschema.FindCatalogEntry(name); !ok {
		return ErrValidationf(fmt.Sprintf("integration schema %q not found in catalog", name))
	}
	if net := strings.TrimSpace(step.AffiliateNetwork); net != "" {
		if _, ok := integrationschema.FindCatalogEntry(net); !ok {
			return ErrValidationf(fmt.Sprintf("affiliate network schema %q not found in catalog", net))
		}
	}
	return nil
}

func validateWizardFlowSkeletonStep(step CampaignWizardFlowSkeletonStep) error {
	if strings.TrimSpace(step.FlowName) == "" {
		return ErrValidationf("flow_name is required")
	}
	if strings.TrimSpace(step.Lander.Name) == "" {
		return ErrValidationf("lander.name is required")
	}
	if strings.TrimSpace(step.Lander.URL) == "" {
		return ErrValidationf("lander.url is required")
	}
	if strings.TrimSpace(step.Offer.Name) == "" || strings.TrimSpace(step.Offer.URL) == "" {
		return ErrValidationf("offer.name and offer.url are required")
	}
	return nil
}

func ValidateWizardTrafficSourceStep(step CampaignWizardTrafficSourceStep) error {
	return validateWizardTrafficSourceStep(step)
}

func ValidateWizardIntegrationTemplateStep(step CampaignWizardIntegrationTemplateStep) error {
	return validateWizardIntegrationTemplateStep(step)
}

func ValidateWizardFlowSkeletonStep(step CampaignWizardFlowSkeletonStep) error {
	return validateWizardFlowSkeletonStep(step)
}

func validateWizardBudgetStep(step CampaignWizardBudgetStep) error {
	if step.BudgetLimitMicro <= 0 {
		return ErrValidationf("budget_limit_micro must be positive")
	}
	return nil
}

func ValidateWizardBudgetStep(step CampaignWizardBudgetStep) error {
	return validateWizardBudgetStep(step)
}
