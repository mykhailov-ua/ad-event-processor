package marginguard

import (
	"context"
	"fmt"
	"strings"
	"time"

	"ad-event-processor/internal/campaign"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/ledger"
	"ad-event-processor/pkg/coldpath"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
	host Host
}

func NewStore(pool *pgxpool.Pool, host Host) *Store {
	return &Store{pool: pool, host: host}
}

func (st *Store) poolOrNil() *pgxpool.Pool {
	if st == nil {
		return nil
	}
	return st.pool
}

func costOverRevenueThresholdBps(policy *ledger.Policy, defaultBps int) int {
	if policy != nil && policy.CostOverRevenueThresholdBps > 0 {
		return policy.CostOverRevenueThresholdBps
	}
	if defaultBps > 0 {
		return defaultBps
	}
	return 500
}

func (st *Store) CreatePolicy(ctx context.Context, p *ledger.Policy) error {
	if st.poolOrNil() == nil || p == nil {
		return fmt.Errorf("service unavailable")
	}
	thresholdBps := costOverRevenueThresholdBps(p, st.host.DefaultCostOverRevenueThresholdBps())
	_, err := st.pool.Exec(ctx, `
		INSERT INTO margin_guard_policies (campaign_id, name, min_clicks, roi_floor_pct, zero_conv_streak, cost_over_revenue_threshold_bps, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, domain.ToUUID(p.CampaignID), p.Name, p.MinClicks, p.RoiFloorPct, p.ZeroConvStreak, thresholdBps, p.IsActive)
	return err
}

func (st *Store) ListPolicies(ctx context.Context, campaignID uuid.UUID) ([]*ledger.Policy, error) {
	if st.poolOrNil() == nil {
		return nil, fmt.Errorf("service unavailable")
	}
	rows, err := st.pool.Query(ctx, `
		SELECT id, campaign_id, name, min_clicks, roi_floor_pct, zero_conv_streak, cost_over_revenue_threshold_bps, is_active
		FROM margin_guard_policies
		WHERE campaign_id = $1
	`, domain.ToUUID(campaignID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var policies []*ledger.Policy
	for rows.Next() {
		p := &ledger.Policy{}
		if err := rows.Scan(&p.ID, &p.CampaignID, &p.Name, &p.MinClicks, &p.RoiFloorPct, &p.ZeroConvStreak, &p.CostOverRevenueThresholdBps, &p.IsActive); err != nil {
			return nil, err
		}
		policies = append(policies, p)
	}
	return policies, nil
}

func (st *Store) GetCampaignMargin(ctx context.Context, campaignID uuid.UUID) (CampaignMargin, error) {
	if st.poolOrNil() == nil {
		return CampaignMargin{}, fmt.Errorf("service unavailable")
	}
	windowStart := time.Now().Add(-1 * time.Hour)
	q := db.New(st.pool)
	sums, err := q.SumCampaignMarginWindow(ctx, db.SumCampaignMarginWindowParams{
		CampaignID: domain.ToUUID(campaignID),
		CreatedAt:  pgtype.Timestamp{Time: windowStart, Valid: true},
	})
	if err != nil {
		return CampaignMargin{}, err
	}
	thresholdBps := costOverRevenueThresholdBps(nil, st.host.DefaultCostOverRevenueThresholdBps())
	policies, err := st.ListPolicies(ctx, campaignID)
	if err != nil {
		return CampaignMargin{}, err
	}
	if len(policies) > 0 {
		thresholdBps = costOverRevenueThresholdBps(policies[0], st.host.DefaultCostOverRevenueThresholdBps())
	}
	limitMicro := ledger.CostOverRevenueLimitMicro(sums.AdvertiserSpendMicro, thresholdBps)
	return CampaignMargin{
		CampaignID:           campaignID.String(),
		WindowStart:          windowStart.UTC().Format(time.RFC3339),
		WindowHours:          1,
		AdvertiserSpendMicro: sums.AdvertiserSpendMicro,
		RtbCostMicro:         sums.RtbCostMicro,
		OperatorMarginMicro:  sums.OperatorMarginMicro,
		PublisherPayoutMicro: sums.PublisherPayoutMicro,
		CostOverRevenueLimit: limitMicro,
		ThresholdBps:         thresholdBps,
		MarginBreach:         sums.RtbCostMicro > limitMicro && sums.AdvertiserSpendMicro > 0,
	}, nil
}

func (st *Store) AttachCampaignListMarginBreach(ctx context.Context, items []campaign.CampaignDTO) {
	if st == nil || len(items) == 0 {
		return
	}
	activeIDs, activeIdx := activeCampaignIDsFromDTOs(items)
	if len(activeIDs) == 0 {
		return
	}
	breaches, err := st.BatchMarginBreach(ctx, activeIDs)
	if err != nil {
		return
	}
	for i, id := range activeIDs {
		items[activeIdx[i]].MarginBreach = breaches[id]
	}
}

func (st *Store) ListActivity(ctx context.Context, campaignID uuid.UUID) ([]ActivityRow, error) {
	if st.poolOrNil() == nil {
		return nil, fmt.Errorf("service unavailable")
	}
	rows, err := st.pool.Query(ctx, `
		SELECT id, policy_id, campaign_id, placement_id, action, reason, metrics, created_at
		FROM margin_guard_activity
		WHERE campaign_id = $1
		ORDER BY created_at DESC
		LIMIT 100
	`, domain.ToUUID(campaignID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var activities []ActivityRow
	for rows.Next() {
		var row ActivityRow
		var metrics []byte
		var createdAt time.Time
		if err := rows.Scan(&row.ID, &row.PolicyID, &row.CampaignID, &row.PlacementID, &row.Action, &row.Reason, &metrics, &createdAt); err != nil {
			return nil, err
		}
		if len(metrics) > 0 {
			row.Metrics = metrics
		}
		row.CreatedAt = createdAt
		activities = append(activities, row)
	}
	return activities, nil
}

func (st *Store) RemovePlacementOverride(ctx context.Context, campaignID uuid.UUID, placementID string) error {
	return st.enqueuePausePlacement(ctx, campaignID, placementID, "remove")
}

func (st *Store) BlockCampaignPlacement(ctx context.Context, campaignID uuid.UUID, placementID string) error {
	placementID = strings.TrimSpace(placementID)
	if placementID == "" {
		return fmt.Errorf("placement_id required")
	}
	return st.enqueuePausePlacement(ctx, campaignID, placementID, "")
}

func (st *Store) enqueuePausePlacement(ctx context.Context, campaignID uuid.UUID, placementID, action string) error {
	if st.poolOrNil() == nil {
		return fmt.Errorf("service unavailable")
	}
	payload, err := coldpath.MarshalOutbox(pausePlacementOutboxPayload{
		CampaignID:  campaignID.String(),
		PlacementID: placementID,
		Action:      action,
	})
	if err != nil {
		return err
	}
	_, err = st.pool.Exec(ctx, `
		INSERT INTO outbox_events (event_type, payload)
		VALUES ($1, $2)`, "PAUSE_PLACEMENT", payload)
	return err
}

type marginSums struct {
	advertiserSpendMicro int64
	rtbCostMicro         int64
}

func (st *Store) BatchMarginBreach(ctx context.Context, campaignIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
	out := make(map[uuid.UUID]bool, len(campaignIDs))
	if st.poolOrNil() == nil || len(campaignIDs) == 0 {
		return out, nil
	}

	pgIDs := make([]pgtype.UUID, len(campaignIDs))
	for i, id := range campaignIDs {
		pgIDs[i] = domain.ToUUID(id)
	}

	windowStart := time.Now().Add(-1 * time.Hour)
	q := db.New(st.pool)

	sumsByCampaign := make(map[uuid.UUID]marginSums, len(campaignIDs))
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
		sumsByCampaign[id] = marginSums{
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

	defaultThreshold := costOverRevenueThresholdBps(nil, st.host.DefaultCostOverRevenueThresholdBps())
	for _, id := range campaignIDs {
		sums := sumsByCampaign[id]
		thresholdBps := defaultThreshold
		if p := policyByCampaign[id]; p != nil {
			thresholdBps = costOverRevenueThresholdBps(p, st.host.DefaultCostOverRevenueThresholdBps())
		}
		limitMicro := ledger.CostOverRevenueLimitMicro(sums.advertiserSpendMicro, thresholdBps)
		out[id] = sums.rtbCostMicro > limitMicro && sums.advertiserSpendMicro > 0
	}
	return out, nil
}

func activeCampaignIDsFromDTOs(items []campaign.CampaignDTO) ([]uuid.UUID, []int) {
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
