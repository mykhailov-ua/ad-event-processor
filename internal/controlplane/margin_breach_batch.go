package controlplane

import (
	"context"
	"time"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/ledger"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type campaignMarginSums struct {
	advertiserSpendMicro int64
	rtbCostMicro         int64
}

func (s *Service) batchCampaignMarginBreach(ctx context.Context, campaignIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
	out := make(map[uuid.UUID]bool, len(campaignIDs))
	if s == nil || s.pool == nil || len(campaignIDs) == 0 {
		return out, nil
	}

	pgIDs := make([]pgtype.UUID, len(campaignIDs))
	for i, id := range campaignIDs {
		pgIDs[i] = domain.ToUUID(id)
	}

	windowStart := time.Now().Add(-1 * time.Hour)
	q := db.New(s.pool)

	sumsByCampaign := make(map[uuid.UUID]campaignMarginSums, len(campaignIDs))
	rows, err := q.SumCampaignMarginWindowByCampaignIDs(ctx, db.SumCampaignMarginWindowByCampaignIDsParams{
		CampaignIds: pgIDs,
		WindowStart: pgtype.Timestamp{Time: windowStart, Valid: true},
	})
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		id, err := uuid.FromBytes(row.CampaignID.Bytes[:])
		if err != nil {
			continue
		}
		sumsByCampaign[id] = campaignMarginSums{
			advertiserSpendMicro: row.AdvertiserSpendMicro,
			rtbCostMicro:         row.RtbCostMicro,
		}
	}

	policyByCampaign := make(map[uuid.UUID]*ledger.Policy, len(campaignIDs))
	policyRows, err := q.ListMarginGuardPoliciesByCampaignIDs(ctx, pgIDs)
	if err != nil {
		return nil, err
	}
	for _, row := range policyRows {
		id, err := uuid.FromBytes(row.CampaignID.Bytes[:])
		if err != nil {
			continue
		}
		if _, exists := policyByCampaign[id]; exists {
			continue
		}
		policyByCampaign[id] = &ledger.Policy{
			ID:                          uuid.UUID(row.ID.Bytes),
			CampaignID:                  id,
			Name:                        row.Name,
			MinClicks:                   int(row.MinClicks),
			RoiFloorPct:                 row.RoiFloorPct,
			ZeroConvStreak:              int(row.ZeroConvStreak),
			CostOverRevenueThresholdBps: int(row.CostOverRevenueThresholdBps),
			IsActive:                    row.IsActive,
		}
	}

	defaultThreshold := ledger.CostOverRevenueThresholdBps(nil, s.cfg)
	for _, id := range campaignIDs {
		sums := sumsByCampaign[id]
		thresholdBps := defaultThreshold
		if p := policyByCampaign[id]; p != nil {
			thresholdBps = ledger.CostOverRevenueThresholdBps(p, s.cfg)
		}
		limitMicro := ledger.CostOverRevenueLimitMicro(sums.advertiserSpendMicro, thresholdBps)
		out[id] = sums.rtbCostMicro > limitMicro && sums.advertiserSpendMicro > 0
	}
	return out, nil
}

func activeCampaignIDsFromDTOs(items []CampaignDTO) ([]uuid.UUID, []int) {
	activeIDs := make([]uuid.UUID, 0, len(items))
	activeIdx := make([]int, 0, len(items))
	for i := range items {
		if items[i].Status != "ACTIVE" {
			continue
		}
		campID, err := uuid.Parse(items[i].ID)
		if err != nil {
			continue
		}
		activeIDs = append(activeIDs, campID)
		activeIdx = append(activeIdx, i)
	}
	return activeIDs, activeIdx
}
