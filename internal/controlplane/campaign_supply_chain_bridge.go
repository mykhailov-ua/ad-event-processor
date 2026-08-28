package controlplane

import (
	"context"
	"encoding/json"

	"ad-event-processor/internal/campaign"
	db "ad-event-processor/internal/domain/db"

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
	b.svc.AuditLog(ctx, q, uid, "UPDATE_CAMPAIGN_SUPPLY_CHAIN", "campaign", &campaignID, auditSupplyChainChange{
		OldNodes: json.RawMessage(oldNodesJSON),
		NewNodes: json.RawMessage(newNodesJSON),
	}, nil)
}

func (s *Service) GetCampaignSupplyChain(ctx context.Context, campaignID uuid.UUID) (CampaignSupplyChainDTO, error) {
	return campaign.GetCampaignSupplyChain(ctx, s.pool, supplyChainBridge{svc: s}, campaignID)
}

func (s *Service) UpdateCampaignSupplyChain(ctx context.Context, campaignID uuid.UUID, nodes []SupplyChainNode) (CampaignSupplyChainDTO, error) {
	return campaign.UpdateCampaignSupplyChain(ctx, s.pool, supplyChainBridge{svc: s}, campaignID, nodes)
}
