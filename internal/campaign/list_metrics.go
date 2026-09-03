package campaign

import (
	"context"
	"fmt"
	"strings"
	"time"

	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/ledger"
	"ad-event-processor/internal/reports"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const campaignListMetricsMaxIDs = 100 // max comma-separated ids= on GET .../campaigns/metrics/batch
const campaignListMetricsCHTimeout = 5 * time.Second

type CampaignListMetricsRowDTO struct {
	CampaignID           string `json:"campaign_id"`
	Impressions          int64  `json:"impressions"`
	Clicks               int64  `json:"clicks"`
	Conversions          int64  `json:"conversions"`
	UniqueClicks         int64  `json:"unique_clicks,omitempty"`
	Blocks               int64  `json:"blocks,omitempty"`
	LeadsRaw             int64  `json:"leads_raw,omitempty"`
	HoldLeads            int64  `json:"hold_leads,omitempty"`
	RejectedLeads        int64  `json:"rejected_leads,omitempty"`
	LPClicks             int64  `json:"lp_clicks,omitempty"`
	LPViews              int64  `json:"lp_views,omitempty"`
	Bots                 int64  `json:"bots,omitempty"`
	Stale                bool   `json:"stale"`
	AdvertiserSpendMicro int64  `json:"advertiser_spend_micro,omitempty"`
	RtbCostMicro         int64  `json:"rtb_cost_micro,omitempty"`
	OperatorMarginMicro  int64  `json:"operator_margin_micro,omitempty"`
	PublisherPayoutMicro int64  `json:"publisher_payout_micro,omitempty"`
	MarginBreach         bool   `json:"margin_breach,omitempty"`
}

type CampaignListMetricsBatchResponse struct {
	Items map[string]CampaignListMetricsRowDTO `json:"items"`
	From  string                               `json:"from"`
	To    string                               `json:"to"`
	Stale bool                                 `json:"stale"`
}

func ParseCampaignListMetricsIDs(raw string) ([]uuid.UUID, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, invalidQueryError("ids required")
	}
	parts := strings.Split(raw, ",")
	if len(parts) > campaignListMetricsMaxIDs {
		return nil, invalidQueryError(fmt.Sprintf("too many ids (max %d)", campaignListMetricsMaxIDs))
	}
	ids := make([]uuid.UUID, 0, len(parts))
	seen := make(map[uuid.UUID]struct{}, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := uuid.Parse(part)
		if err != nil {
			return nil, invalidQueryError("invalid campaign id in ids")
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil, invalidQueryError("ids required")
	}
	return ids, nil
}

func BatchCampaignListMetrics(
	ctx context.Context,
	pool *pgxpool.Pool,
	clickhouseQuery *database.ClickHouseQuery,
	defaultThresholdBps int,
	campaignIDs []uuid.UUID,
	from, to time.Time,
) (CampaignListMetricsBatchResponse, error) {
	if pool == nil {
		return CampaignListMetricsBatchResponse{}, ErrServiceUnavailable()
	}
	if !to.After(from) {
		return CampaignListMetricsBatchResponse{}, fmt.Errorf("%w: to must be after from", ErrInvalidTimeRange)
	}

	pgIDs := make([]pgtype.UUID, len(campaignIDs))
	for i, id := range campaignIDs {
		pgIDs[i] = domain.ToUUID(id)
	}

	q := db.New(pool)
	statsFrom, statsTo := CampaignListPGStatsDates(from, to)
	marginStart, marginEnd := CampaignListMarginWindow(from, to)
	statsRows, err := q.SumCampaignStatsByCampaignIDsInRange(ctx, db.SumCampaignStatsByCampaignIDsInRangeParams{
		CampaignIds: pgIDs,
		FromDate:    statsFrom,
		ToDate:      statsTo,
	})
	if err != nil {
		return CampaignListMetricsBatchResponse{}, err
	}

	items := make(map[string]CampaignListMetricsRowDTO, len(campaignIDs))
	for _, id := range campaignIDs {
		items[id.String()] = CampaignListMetricsRowDTO{CampaignID: id.String()}
	}
	for _, row := range statsRows {
		id := uuid.UUID(row.CampaignID.Bytes).String()
		entry := items[id]
		entry.Impressions = row.Impressions
		entry.Clicks = row.Clicks
		entry.Conversions = row.Conversions
		items[id] = entry
	}

	// stale stays true until ClickHouse returns sort and funnel aggregates.
	stale := true
	if clickhouseQuery != nil {
		chCtx, cancel := context.WithTimeout(ctx, campaignListMetricsCHTimeout)
		chMetrics, errCH := reports.QueryCampaignListCHSortMetricsCH(chCtx, clickhouseQuery, campaignIDs, from, to)
		funnelMetrics, errFunnel := reports.QueryCampaignListFunnelMetricsCH(chCtx, clickhouseQuery, campaignIDs, from, to)
		cancel()
		if errCH == nil {
			for campaignID, row := range chMetrics {
				entry := items[campaignID]
				entry.UniqueClicks = row.UniqueClicks
				entry.Blocks = row.Blocks
				entry.LPClicks = row.LPClicks
				entry.Bots = row.Bots
				items[campaignID] = entry
			}
		}
		if errFunnel == nil {
			for campaignID, funnel := range funnelMetrics {
				entry := items[campaignID]
				entry.HoldLeads = funnel.HoldLeads
				entry.RejectedLeads = funnel.RejectedLeads
				if funnel.LeadsRaw() > 0 {
					entry.LeadsRaw = funnel.LeadsRaw()
				}
				items[campaignID] = entry
			}
		}
		if errCH == nil && errFunnel == nil {
			stale = false
		}
	}
	for id, entry := range items {
		if entry.LeadsRaw == 0 {
			derived := entry.Conversions + entry.HoldLeads + entry.RejectedLeads
			if derived > 0 {
				entry.LeadsRaw = derived
			} else {
				entry.LeadsRaw = entry.Conversions
			}
		}
		if entry.LPViews == 0 {
			entry.LPViews = entry.Impressions
		}
		items[id] = entry
	}

	marginRows, err := q.SumCampaignMarginWindowByCampaignIDsInRange(ctx, db.SumCampaignMarginWindowByCampaignIDsInRangeParams{
		CampaignIds: pgIDs,
		WindowStart: pgtype.Timestamp{Time: marginStart, Valid: true},
		WindowEnd:   pgtype.Timestamp{Time: marginEnd, Valid: true},
	})
	if err != nil {
		return CampaignListMetricsBatchResponse{}, err
	}

	thresholdBps := defaultThresholdBps
	if thresholdBps <= 0 {
		thresholdBps = 500
	}
	policyByCampaign := map[uuid.UUID]int{}
	policyRows, err := q.ListMarginGuardPoliciesByCampaignIDs(ctx, pgIDs)
	if err != nil {
		return CampaignListMetricsBatchResponse{}, err
	}
	for _, policyRow := range policyRows {
		id, parseErr := uuid.FromBytes(policyRow.CampaignID.Bytes[:])
		if parseErr != nil {
			continue
		}
		if _, seen := policyByCampaign[id]; seen {
			continue
		}
		bps := policyRow.CostOverRevenueThresholdBps
		if bps <= 0 {
			bps = int32(thresholdBps)
		}
		policyByCampaign[id] = int(bps)
	}

	for _, row := range marginRows {
		id := uuid.UUID(row.CampaignID.Bytes).String()
		entry := items[id]
		entry.AdvertiserSpendMicro = row.AdvertiserSpendMicro
		entry.RtbCostMicro = row.RtbCostMicro
		entry.OperatorMarginMicro = row.OperatorMarginMicro
		entry.PublisherPayoutMicro = row.PublisherPayoutMicro
		campaignUUID, _ := uuid.Parse(id)
		bps := thresholdBps
		if custom, ok := policyByCampaign[campaignUUID]; ok {
			bps = custom
		}
		limitMicro := ledger.CostOverRevenueLimitMicro(row.AdvertiserSpendMicro, bps)
		entry.MarginBreach = row.RtbCostMicro > limitMicro && row.AdvertiserSpendMicro > 0
		items[id] = entry
	}

	return CampaignListMetricsBatchResponse{
		Items: items,
		From:  from.UTC().Format(time.RFC3339),
		To:    to.UTC().Format(time.RFC3339),
		Stale: stale,
	}, nil
}

func AttachCampaignListMarginBreachInRange(
	ctx context.Context,
	pool *pgxpool.Pool,
	defaultThresholdBps int,
	items []CampaignDTO,
	from, to time.Time,
) {
	if pool == nil || len(items) == 0 {
		return
	}
	activeIDs, activeIdx := activeCampaignListIndices(items)
	if len(activeIDs) == 0 {
		return
	}

	pgIDs := make([]pgtype.UUID, len(activeIDs))
	for i, id := range activeIDs {
		pgIDs[i] = domain.ToUUID(id)
	}

	marginStart, marginEnd := CampaignListMarginWindow(from, to)
	q := db.New(pool)
	marginRows, err := q.SumCampaignMarginWindowByCampaignIDsInRange(ctx, db.SumCampaignMarginWindowByCampaignIDsInRangeParams{
		CampaignIds: pgIDs,
		WindowStart: pgtype.Timestamp{Time: marginStart, Valid: true},
		WindowEnd:   pgtype.Timestamp{Time: marginEnd, Valid: true},
	})
	if err != nil {
		return
	}

	thresholdBps := defaultThresholdBps
	if thresholdBps <= 0 {
		thresholdBps = 500
	}
	policyByCampaign := map[uuid.UUID]int{}
	policyRows, err := q.ListMarginGuardPoliciesByCampaignIDs(ctx, pgIDs)
	if err != nil {
		return
	}
	for _, policyRow := range policyRows {
		id, parseErr := uuid.FromBytes(policyRow.CampaignID.Bytes[:])
		if parseErr != nil {
			continue
		}
		if _, seen := policyByCampaign[id]; seen {
			continue
		}
		bps := policyRow.CostOverRevenueThresholdBps
		if bps <= 0 {
			bps = int32(thresholdBps)
		}
		policyByCampaign[id] = int(bps)
	}

	breachByID := make(map[uuid.UUID]bool, len(activeIDs))
	for _, row := range marginRows {
		id := uuid.UUID(row.CampaignID.Bytes)
		bps := thresholdBps
		if custom, ok := policyByCampaign[id]; ok {
			bps = custom
		}
		limitMicro := ledger.CostOverRevenueLimitMicro(row.AdvertiserSpendMicro, bps)
		breachByID[id] = row.RtbCostMicro > limitMicro && row.AdvertiserSpendMicro > 0
	}
	for i, id := range activeIDs {
		items[activeIdx[i]].MarginBreach = breachByID[id]
	}
}

func activeCampaignListIndices(items []CampaignDTO) ([]uuid.UUID, []int) {
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
