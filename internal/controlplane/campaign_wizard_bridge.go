package controlplane

import (
	"context"
	"fmt"
	"strings"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/shardadmin"

	"ad-event-processor/internal/campaign"
	"ad-event-processor/internal/campaign/integration"
	"ad-event-processor/internal/campaign/wizard"
	"ad-event-processor/internal/integrationschema"

	"github.com/google/uuid"
)

const (
	wizardStepTrafficSource       = wizard.WizardStepTrafficSource
	wizardStepIntegrationTemplate = wizard.WizardStepIntegrationTemplate
	wizardStepFlowSkeleton        = wizard.WizardStepFlowSkeleton
	wizardStepBudget              = wizard.WizardStepBudget
	wizardStepReview              = wizard.WizardStepReview
)

func (s *Service) WizardStore() *wizard.WizardStore {
	if s == nil {
		return nil
	}
	if s.wizardStore == nil {
		s.wizardStore = wizard.NewWizardStore(s.pool, s)
	}
	return s.wizardStore
}

func (s *Service) CreateCampaignWizardSession(ctx context.Context, customerID uuid.UUID, templateKey string) (campaign.CampaignWizardSessionDTO, error) {
	return s.WizardStore().CreateCampaignWizardSession(ctx, customerID, templateKey)
}

func (s *Service) GetCampaignWizardSession(ctx context.Context, sessionID uuid.UUID) (campaign.CampaignWizardSessionDTO, error) {
	return s.WizardStore().GetCampaignWizardSession(ctx, sessionID)
}

func (s *Service) UpdateCampaignWizardSessionStep(ctx context.Context, sessionID uuid.UUID, step string, payload []byte) (campaign.CampaignWizardSessionDTO, error) {
	return s.WizardStore().UpdateCampaignWizardSessionStep(ctx, sessionID, step, payload)
}

func (s *Service) CommitCampaignWizardSession(ctx context.Context, sessionID uuid.UUID, idempotencyKey string, publish bool) (campaign.CampaignWizardCommitResult, error) {
	return s.WizardStore().CommitCampaignWizardSession(ctx, sessionID, idempotencyKey, publish)
}

func (s *Service) ApplyOnboardingTemplate(key string) (campaign.CampaignWizardStored, error) {
	return campaign.ApplyOnboardingTemplate(key)
}

func (s *Service) ImportBundledTemplate(ctx context.Context, schemaName string) error {
	entry, ok := integrationschema.FindCatalogEntry(schemaName)
	if !ok {
		return errValidation(fmt.Sprintf("integration schema %q not found in catalog", schemaName))
	}
	_, err := s.TemplateCatalog(s.pool).ImportCatalogEntry(ctx, entry)
	return err
}

func (s *Service) ApplyAffiliateNetworkTemplate(ctx context.Context, campaignID uuid.UUID, network, trackingDomain string) error {
	_, err := s.ApplyCampaignTemplates(ctx, campaignID, campaign.ApplyCampaignTemplatesRequest{
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

func (s *Service) AuthorizeCampaignSmoke(ctx context.Context, campaignID uuid.UUID) error {
	row, err := s.GetCampaignRow(ctx, campaignID)
	if err != nil {
		return err
	}
	return campaign.AssertMediaBuyerCampaignAccess(ctx, row)
}

func (s *Service) TrackerPublicBaseURL() string {
	if s.cfg != nil {
		return strings.TrimSpace(s.cfg.LanderPublicBaseURL)
	}
	return ""
}

func (s *Service) EvaluateCampaignPublish(ctx context.Context, campaignID uuid.UUID) (campaign.CampaignPublishCheckDTO, error) {
	return s.CampaignRuntime().EvaluateCampaignPublish(ctx, campaignID)
}

func (s *Service) PublishCampaign(ctx context.Context, campaignID uuid.UUID, force bool) (campaign.CampaignDTO, error) {
	return s.CampaignRuntime().PublishCampaign(ctx, campaignID, force)
}

func (s *Service) GetCampaignIntegrationHealth(ctx context.Context, campaignID uuid.UUID) (campaign.IntegrationHealthDTO, error) {
	return integration.GetCampaignIntegrationHealth(ctx, s.pool, s, campaignID)
}

func (s *Service) ListCampaignConversionMappings(ctx context.Context, campaignID uuid.UUID) ([]campaign.ConversionMappingDTO, error) {
	return campaign.ListCampaignConversionMappings(ctx, s.pool, campaignID)
}

func (s *Service) ReplaceCampaignConversionMappings(ctx context.Context, campaignID uuid.UUID, mappings []campaign.ConversionMappingDTO) ([]campaign.ConversionMappingDTO, error) {
	return campaign.ReplaceCampaignConversionMappings(ctx, s.pool, campaignID, mappings)
}

func (s *Service) AuditCampaignRevisionConflict(ctx context.Context, campaignID uuid.UUID, expectedRevision string) {
	integration.AuditCampaignRevisionConflict(ctx, s.pool, s, campaignID, expectedRevision)
}

func (s *Service) cloneCampaignPlacementBlocks(ctx context.Context, sourceID, destID uuid.UUID) error {
	if s == nil || len(s.redisShards) == 0 {
		return nil
	}
	redisClient := s.redisClientForCampaign(sourceID)
	if redisClient == nil {
		return nil
	}
	key := domain.PlacementBlacklistKey(sourceID)
	placements, err := redisClient.HKeys(ctx, key).Result()
	if err != nil {
		return err
	}
	if len(placements) == 0 {
		return nil
	}
	destKey := domain.PlacementBlacklistKey(destID)
	for _, placementID := range placements {
		placementID = strings.TrimSpace(placementID)
		if placementID == "" {
			continue
		}
		if err := shardadmin.SyncGlobalHashFieldToAllShards(ctx, s.redisShards, destKey, placementID, "1", false); err != nil {
			return err
		}
	}
	return nil
}
