package controlplane

import (
	"ad-event-processor/internal/campaign"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/integrationschema"
	"ad-event-processor/internal/platformadmin"
	"ad-event-processor/internal/supply"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type supplyChainBridge struct {
	svc *Service
}

func (b supplyChainBridge) MapCampaignNotFound(err error) error {
	return mapNotFound(err, ErrCampaignNotFound)
}

func (b supplyChainBridge) AuditSupplyChainUpdate(ctx context.Context, q db.Querier, campaignID uuid.UUID, oldNodesJSON, newNodesJSON []byte) {
	var uid uuid.UUID
	if u, ok := GetUser(ctx); ok {
		uid = u.UserID
	}
	b.svc.AuditLog(ctx, q, uid, "UPDATE_CAMPAIGN_SUPPLY_CHAIN", "campaign", &campaignID, platformadmin.AuditSupplyChainChange{
		OldNodes: json.RawMessage(oldNodesJSON),
		NewNodes: json.RawMessage(newNodesJSON),
	}, nil)
}

func (s *Service) GetCampaignSupplyChain(ctx context.Context, campaignID uuid.UUID) (supply.CampaignChainDTO, error) {
	return campaign.GetCampaignSupplyChain(ctx, s.pool, supplyChainBridge{svc: s}, campaignID)
}

func (s *Service) UpdateCampaignSupplyChain(ctx context.Context, campaignID uuid.UUID, nodes []supply.ChainNode) (supply.CampaignChainDTO, error) {
	return campaign.UpdateCampaignSupplyChain(ctx, s.pool, supplyChainBridge{svc: s}, campaignID, nodes)
}

func (s *Service) WizardStore() *campaign.WizardStore {
	if s == nil {
		return nil
	}
	if s.wizardStore == nil {
		s.wizardStore = campaign.NewWizardStore(s.pool, s)
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
