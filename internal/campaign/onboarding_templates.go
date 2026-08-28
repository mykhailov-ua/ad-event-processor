package campaign

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

type OnboardingFlowDefault struct {
	FlowName string                 `json:"flow_name"`
	Lander   CampaignWizardAssetRef `json:"lander"`
	Offer    CampaignWizardAssetRef `json:"offer"`
}

type OnboardingTemplate struct {
	Key                   string                `json:"key"`
	Title                 string                `json:"title"`
	Description           string                `json:"description"`
	TrafficFamily         string                `json:"traffic_family"`
	DefaultFlow           OnboardingFlowDefault `json:"default_flow"`
	IntegrationSchemaRefs []string              `json:"integration_schema_refs"`
	SampleMacros          map[string]string     `json:"sample_macros"`
}

type onboardingTemplateWizardYAML struct {
	TrafficTemplateID string            `yaml:"traffic_template_id"`
	IntegrationSchema string            `yaml:"integration_schema"`
	ClickQueryParams  map[string]string `yaml:"click_query_params"`
	CampaignName      string            `yaml:"campaign_name"`
	FlowName          string            `yaml:"flow_name"`
	LanderName        string            `yaml:"lander_name"`
	LanderURL         string            `yaml:"lander_url"`
	OfferName         string            `yaml:"offer_name"`
	OfferURL          string            `yaml:"offer_url"`
	BudgetLimitMicro  int64             `yaml:"budget_limit_micro"`
	Timezone          string            `yaml:"timezone"`
	TargetCountries   []string          `yaml:"target_countries"`
}

type onboardingTemplateYAML struct {
	Key                   string                       `yaml:"key"`
	Title                 string                       `yaml:"title"`
	Description           string                       `yaml:"description"`
	TrafficFamily         string                       `yaml:"traffic_family"`
	IntegrationSchemaRefs []string                     `yaml:"integration_schema_refs"`
	SampleMacros          map[string]string            `yaml:"sample_macros"`
	Wizard                onboardingTemplateWizardYAML `yaml:"wizard"`
}

type onboardingCatalogYAML struct {
	Version   int                      `yaml:"version"`
	Templates []onboardingTemplateYAML `yaml:"templates"`
}

var (
	onboardingCatalogOnce sync.Once
	onboardingCatalogErr  error
	onboardingCatalog     []onboardingTemplateDef
)

type onboardingTemplateDef struct {
	OnboardingTemplate
	wizard onboardingTemplateWizardYAML
}

func ListOnboardingTemplates() ([]OnboardingTemplate, error) {
	defs, err := loadOnboardingCatalog()
	if err != nil {
		return nil, err
	}
	out := make([]OnboardingTemplate, 0, len(defs))
	for _, def := range defs {
		out = append(out, def.OnboardingTemplate)
	}
	return out, nil
}

func OnboardingTemplateKeys() ([]string, error) {
	defs, err := loadOnboardingCatalog()
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(defs))
	for _, def := range defs {
		out = append(out, def.Key)
	}
	return out, nil
}

func ApplyOnboardingTemplate(key string) (CampaignWizardStored, error) {
	key = strings.TrimSpace(key)
	defs, err := loadOnboardingCatalog()
	if err != nil {
		return CampaignWizardStored{}, err
	}
	keys, _ := OnboardingTemplateKeys()
	for _, def := range defs {
		if def.Key != key {
			continue
		}
		w := def.wizard
		stored := CampaignWizardStored{
			TrafficSource: CampaignWizardTrafficSourceStep{
				Name:              strings.TrimSpace(w.CampaignName),
				TrafficTemplateID: strings.TrimSpace(w.TrafficTemplateID),
				ClickQueryParams:  w.ClickQueryParams,
			},
			IntegrationTemplate: CampaignWizardIntegrationTemplateStep{
				IntegrationSchema: strings.TrimSpace(w.IntegrationSchema),
			},
			FlowSkeleton: CampaignWizardFlowSkeletonStep{
				FlowName: strings.TrimSpace(w.FlowName),
				Lander: CampaignWizardAssetRef{
					Name: strings.TrimSpace(w.LanderName),
					URL:  strings.TrimSpace(w.LanderURL),
				},
				Offer: CampaignWizardAssetRef{
					Name: strings.TrimSpace(w.OfferName),
					URL:  strings.TrimSpace(w.OfferURL),
				},
			},
			Budget: CampaignWizardBudgetStep{
				BudgetLimitMicro: w.BudgetLimitMicro,
				Timezone:         strings.TrimSpace(w.Timezone),
				TargetCountries:  append([]string(nil), w.TargetCountries...),
			},
		}
		if err := validateOnboardingWizardStored(stored); err != nil {
			return CampaignWizardStored{}, err
		}
		return stored, nil
	}
	return CampaignWizardStored{}, errValidation(fmt.Sprintf("unknown template_key %q; valid keys: %s", key, strings.Join(keys, ", ")))
}

func validateOnboardingWizardStored(stored CampaignWizardStored) error {
	if err := validateWizardTrafficSourceStep(stored.TrafficSource); err != nil {
		return err
	}
	if err := validateWizardIntegrationTemplateStep(stored.IntegrationTemplate); err != nil {
		return err
	}
	if err := validateWizardFlowSkeletonStep(stored.FlowSkeleton); err != nil {
		return err
	}
	if err := validateWizardBudgetStep(stored.Budget); err != nil {
		return err
	}
	return nil
}

func loadOnboardingCatalog() ([]onboardingTemplateDef, error) {
	onboardingCatalogOnce.Do(func() {
		path, err := resolveOnboardingCatalogPath()
		if err != nil {
			onboardingCatalogErr = err
			return
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			onboardingCatalogErr = fmt.Errorf("read onboarding catalog: %w", err)
			return
		}
		var parsed onboardingCatalogYAML
		if err := yaml.Unmarshal(raw, &parsed); err != nil {
			onboardingCatalogErr = fmt.Errorf("parse onboarding catalog: %w", err)
			return
		}
		if parsed.Version != 1 {
			onboardingCatalogErr = fmt.Errorf("unsupported onboarding catalog version %d", parsed.Version)
			return
		}
		defs := make([]onboardingTemplateDef, 0, len(parsed.Templates))
		for _, row := range parsed.Templates {
			key := strings.TrimSpace(row.Key)
			if key == "" {
				onboardingCatalogErr = fmt.Errorf("onboarding template missing key")
				return
			}
			defs = append(defs, onboardingTemplateDef{
				OnboardingTemplate: OnboardingTemplate{
					Key:           key,
					Title:         strings.TrimSpace(row.Title),
					Description:   strings.TrimSpace(row.Description),
					TrafficFamily: strings.TrimSpace(row.TrafficFamily),
					DefaultFlow: OnboardingFlowDefault{
						FlowName: strings.TrimSpace(row.Wizard.FlowName),
						Lander: CampaignWizardAssetRef{
							Name: strings.TrimSpace(row.Wizard.LanderName),
							URL:  strings.TrimSpace(row.Wizard.LanderURL),
						},
						Offer: CampaignWizardAssetRef{
							Name: strings.TrimSpace(row.Wizard.OfferName),
							URL:  strings.TrimSpace(row.Wizard.OfferURL),
						},
					},
					IntegrationSchemaRefs: append([]string(nil), row.IntegrationSchemaRefs...),
					SampleMacros:          row.SampleMacros,
				},
				wizard: row.Wizard,
			})
		}
		onboardingCatalog = defs
	})
	return onboardingCatalog, onboardingCatalogErr
}

func resolveOnboardingCatalogPath() (string, error) {
	name := filepath.Join("onboarding", "catalog.v1.yaml")
	if root := strings.TrimSpace(os.Getenv("REPO_ROOT")); root != "" {
		candidate := filepath.Join(root, "deploy", "schemas", name)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	candidates := []string{
		filepath.Join("deploy", "schemas", name),
		filepath.Join("..", "..", "deploy", "schemas", name),
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("onboarding catalog not found")
}

func resetOnboardingCatalogForTest() {
	onboardingCatalogOnce = sync.Once{}
	onboardingCatalogErr = nil
	onboardingCatalog = nil
}
