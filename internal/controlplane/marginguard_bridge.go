package controlplane

import (
	"context"

	"ad-event-processor/internal/ledger"
	"ad-event-processor/internal/marginguard"

	"github.com/google/uuid"
)

var _ marginguard.Host = (*Service)(nil)

func (s *Service) MarginGuardStore() *marginguard.Store {
	if s == nil {
		return nil
	}
	if s.marginGuardStore == nil {
		s.marginGuardStore = marginguard.NewStore(s.pool, s)
	}
	return s.marginGuardStore
}

func (s *Service) DefaultCostOverRevenueThresholdBps() int {
	if s.cfg != nil && s.cfg.MarginGuardDefaultThresholdBps > 0 {
		return s.cfg.MarginGuardDefaultThresholdBps
	}
	return 500
}

func (s *Service) CreateMarginGuardPolicy(ctx context.Context, p *ledger.Policy) error {
	return s.MarginGuardStore().CreatePolicy(ctx, p)
}

func (s *Service) ListMarginGuardPolicies(ctx context.Context, campaignID uuid.UUID) ([]*ledger.Policy, error) {
	return s.MarginGuardStore().ListPolicies(ctx, campaignID)
}

func (s *Service) GetCampaignMargin(ctx context.Context, campaignID uuid.UUID) (CampaignMarginDTO, error) {
	m, err := s.MarginGuardStore().GetCampaignMargin(ctx, campaignID)
	if err != nil {
		return CampaignMarginDTO{}, err
	}
	return campaignMarginToDTO(m), nil
}

func (s *Service) AttachCampaignListMarginBreach(ctx context.Context, items []CampaignDTO) {
	s.MarginGuardStore().AttachCampaignListMarginBreach(ctx, items)
}

func (s *Service) GetMarginGuardActivity(ctx context.Context, campaignID uuid.UUID) ([]MarginGuardActivityRow, error) {
	rows, err := s.MarginGuardStore().ListActivity(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	out := make([]MarginGuardActivityRow, len(rows))
	for i, row := range rows {
		out[i] = MarginGuardActivityRow(row)
	}
	return out, nil
}

func (s *Service) RemovePlacementOverride(ctx context.Context, campaignID uuid.UUID, placementID string) error {
	return s.MarginGuardStore().RemovePlacementOverride(ctx, campaignID, placementID)
}

func (s *Service) BlockCampaignPlacement(ctx context.Context, campaignID uuid.UUID, placementID string) error {
	return s.MarginGuardStore().BlockCampaignPlacement(ctx, campaignID, placementID)
}

func (s *Service) batchCampaignMarginBreach(ctx context.Context, campaignIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
	return s.MarginGuardStore().BatchMarginBreach(ctx, campaignIDs)
}

func campaignMarginToDTO(m marginguard.CampaignMargin) CampaignMarginDTO {
	return CampaignMarginDTO{
		CampaignID:           m.CampaignID,
		WindowStart:          m.WindowStart,
		WindowHours:          m.WindowHours,
		AdvertiserSpendMicro: m.AdvertiserSpendMicro,
		RtbCostMicro:         m.RtbCostMicro,
		OperatorMarginMicro:  m.OperatorMarginMicro,
		PublisherPayoutMicro: m.PublisherPayoutMicro,
		CostOverRevenueLimit: m.CostOverRevenueLimit,
		ThresholdBps:         m.ThresholdBps,
		MarginBreach:         m.MarginBreach,
	}
}

type marginGuardServiceAdapter struct {
	svc *Service
}

func (a marginGuardServiceAdapter) ListPolicies(ctx context.Context, campaignID uuid.UUID) ([]*ledger.Policy, error) {
	return a.svc.ListMarginGuardPolicies(ctx, campaignID)
}

func (a marginGuardServiceAdapter) CreatePolicy(ctx context.Context, p *ledger.Policy) error {
	return a.svc.CreateMarginGuardPolicy(ctx, p)
}

func (a marginGuardServiceAdapter) ListActivity(ctx context.Context, campaignID uuid.UUID) ([]marginguard.ActivityRow, error) {
	rows, err := a.svc.GetMarginGuardActivity(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	out := make([]marginguard.ActivityRow, len(rows))
	for i, row := range rows {
		out[i] = marginguard.ActivityRow(row)
	}
	return out, nil
}

func (a marginGuardServiceAdapter) RemovePlacementOverride(ctx context.Context, campaignID uuid.UUID, placementID string) error {
	return a.svc.RemovePlacementOverride(ctx, campaignID, placementID)
}
