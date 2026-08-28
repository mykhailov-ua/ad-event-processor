package controlplane

import (
	"context"
	"fmt"
	"strings"

	"ad-event-processor/internal/campaign"
	"ad-event-processor/internal/integrationschema"

	"github.com/google/uuid"
)

var (
	errCampaignWizardSessionNotFound = campaign.ErrCampaignWizardSessionNotFound
	errCampaignWizardSessionExpired  = campaign.ErrCampaignWizardSessionExpired
	errCampaignWizardIncomplete      = campaign.ErrCampaignWizardIncomplete
)

const (
	wizardStepTrafficSource       = campaign.WizardStepTrafficSource
	wizardStepIntegrationTemplate = campaign.WizardStepIntegrationTemplate
	wizardStepFlowSkeleton        = campaign.WizardStepFlowSkeleton
	wizardStepBudget              = campaign.WizardStepBudget
	wizardStepReview              = campaign.WizardStepReview
)

var _ campaign.WizardHost = (*Service)(nil)

func (s *Service) WizardStore() *campaign.WizardStore {
	if s == nil {
		return nil
	}
	if s.wizardStore == nil {
		s.wizardStore = campaign.NewWizardStore(s.pool, s)
	}
	return s.wizardStore
}

func (s *Service) CreateCampaignWizardSession(ctx context.Context, customerID uuid.UUID, templateKey string) (CampaignWizardSessionDTO, error) {
	return s.WizardStore().CreateCampaignWizardSession(ctx, customerID, templateKey)
}

func (s *Service) GetCampaignWizardSession(ctx context.Context, sessionID uuid.UUID) (CampaignWizardSessionDTO, error) {
	return s.WizardStore().GetCampaignWizardSession(ctx, sessionID)
}

func (s *Service) UpdateCampaignWizardSessionStep(ctx context.Context, sessionID uuid.UUID, step string, payload []byte) (CampaignWizardSessionDTO, error) {
	return s.WizardStore().UpdateCampaignWizardSessionStep(ctx, sessionID, step, payload)
}

func (s *Service) CommitCampaignWizardSession(ctx context.Context, sessionID uuid.UUID, idempotencyKey string, publish bool) (CampaignWizardCommitResult, error) {
	return s.WizardStore().CommitCampaignWizardSession(ctx, sessionID, idempotencyKey, publish)
}

func (s *Service) ApplyOnboardingTemplate(key string) (CampaignWizardStored, error) {
	return campaign.ApplyOnboardingTemplate(key)
}

func (s *Service) ImportBundledTemplate(ctx context.Context, schemaName string) error {
	entry, ok := integrationschema.FindCatalogEntry(schemaName)
	if !ok {
		return errValidation(fmt.Sprintf("integration schema %q not found in catalog", schemaName))
	}
	_, err := s.importBundledTemplate(ctx, entry)
	return err
}

func (s *Service) ApplyAffiliateNetworkTemplate(ctx context.Context, campaignID uuid.UUID, network, trackingDomain string) error {
	_, err := s.ApplyCampaignTemplates(ctx, campaignID, ApplyCampaignTemplatesRequest{
		AffiliateNetwork: network,
		TrackingDomain:   trackingDomain,
	})
	return err
}

func (s *Service) TrackingDomain(ctx context.Context, override string) string {
	if d := strings.TrimSpace(override); d != "" {
		return d
	}
	if cfg, _, err := s.GetPlatformConfig(ctx); err == nil {
		if d := strings.TrimSpace(cfg.TrackingDomain); d != "" {
			return d
		}
	}
	if s.cfg != nil {
		return strings.TrimSpace(s.cfg.LanderPublicBaseURL)
	}
	return ""
}

func (s *Service) InboundTargetURL(ctx context.Context, schemaName, trackingDomain string) (string, error) {
	schemaName = strings.TrimSpace(schemaName)
	if schemaName == "" {
		return "", nil
	}
	entry, ok := integrationschema.FindCatalogEntry(schemaName)
	if !ok {
		return "", errValidation(fmt.Sprintf("integration schema %q not found in catalog", schemaName))
	}
	_, kind, parsed, err := integrationschema.LoadBundledTemplate(entry)
	if err != nil {
		return "", errValidation(err.Error())
	}
	if kind != integrationschema.KindInboundTokens {
		return "", nil
	}
	inbound := parsed.(*integrationschema.InboundTokensSchema)
	return integrationschema.BuildInboundTrackingURL(s.TrackingDomain(ctx, trackingDomain), inbound), nil
}
