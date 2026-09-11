package platformadmin

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/reports"
	"ad-event-processor/internal/teamscope"
	"ad-event-processor/pkg/coldpath"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const TeamMetricsByOwnerCap = 50

type TeamPeriodDTO struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type TeamMetricsBlockDTO struct {
	SpendMicro   int64                    `json:"spend_micro"`
	CostMicro    int64                    `json:"cost_micro,omitempty"`
	RevenueMicro int64                    `json:"revenue_micro"`
	ProfitMicro  int64                    `json:"profit_micro"`
	Conversions  int64                    `json:"conversions"`
	UniqueClicks int64                    `json:"unique_clicks,omitempty"`
	CPCMicro     int64                    `json:"cpc_micro,omitempty"`
	CPAMicro     int64                    `json:"cpa_micro"`
	EPCMicro     int64                    `json:"epc_micro,omitempty"`
	CRPct        float64                  `json:"cr_pct,omitempty"`
	ROIPct       float64                  `json:"roi_pct"`
	Freshness    reports.DataFreshnessDTO `json:"freshness"`
}

type TeamOwnerMetricsDTO struct {
	UserID string              `json:"user_id"`
	Email  string              `json:"email,omitempty"`
	KPIs   TeamMetricsBlockDTO `json:"kpis"`
}

type TeamMetricsResponse struct {
	CustomerID string                `json:"customer_id"`
	Period     TeamPeriodDTO         `json:"period"`
	Aggregate  TeamMetricsBlockDTO   `json:"aggregate"`
	ByOwner    []TeamOwnerMetricsDTO `json:"by_owner"`
}

type teamOwnerAccumulator struct {
	clicks       int64
	conversions  int64
	spendMicro   int64
	revenueMicro int64
}

func finalizeTeamMetricsBlock(kpis *TeamMetricsBlockDTO, clicks int64) {
	if kpis == nil {
		return
	}
	costMicro := kpis.CostMicro
	if costMicro == 0 {
		costMicro = kpis.SpendMicro
		kpis.CostMicro = costMicro
	}
	if kpis.ProfitMicro == 0 && kpis.RevenueMicro > 0 && costMicro > 0 {
		kpis.ProfitMicro = kpis.RevenueMicro - costMicro
	}
	if clicks > 0 {
		kpis.CPCMicro = reports.ComputeCPCMicro(costMicro, clicks)
		kpis.EPCMicro = reports.ComputeEPCMicro(kpis.RevenueMicro, clicks)
		kpis.CRPct = reports.ComputeCRPct(kpis.Conversions, clicks)
	}
	if kpis.Conversions > 0 && costMicro > 0 {
		kpis.CPAMicro = reports.ComputeCPAMicro(costMicro, kpis.Conversions)
	}
	if costMicro > 0 {
		kpis.ROIPct = reports.ComputeROIPct(kpis.ProfitMicro, costMicro)
	}
}

func accumulatorToMetricsBlock(acc teamOwnerAccumulator, freshness reports.DataFreshnessDTO) TeamMetricsBlockDTO {
	profitMicro := acc.revenueMicro - acc.spendMicro
	block := TeamMetricsBlockDTO{
		SpendMicro:   acc.spendMicro,
		CostMicro:    acc.spendMicro,
		RevenueMicro: acc.revenueMicro,
		ProfitMicro:  profitMicro,
		Conversions:  acc.conversions,
		Freshness:    freshness,
	}
	finalizeTeamMetricsBlock(&block, acc.clicks)
	return block
}

func (s *TeamOverviewService) GetTeamMetrics(ctx context.Context, customerID uuid.UUID, from, to time.Time) (TeamMetricsResponse, error) {
	if s == nil || s.Pool == nil {
		return TeamMetricsResponse{}, errors.New("team service unavailable")
	}
	if customerID == uuid.Nil {
		return TeamMetricsResponse{}, fmt.Errorf("customer_id required")
	}
	if from.IsZero() {
		from = time.Now().UTC().Add(-7 * 24 * time.Hour)
	}
	if to.IsZero() {
		to = time.Now().UTC()
	}
	if err := reports.ValidateChartRange(from, to); err != nil {
		return TeamMetricsResponse{}, err
	}

	scope, err := teamscope.ResolveListScope(ctx, s.Pool, customerID)
	if err != nil {
		return TeamMetricsResponse{}, err
	}
	campaignOwners, err := loadScopedCampaignOwners(ctx, s.Pool, customerID, scope)
	if err != nil {
		return TeamMetricsResponse{}, err
	}

	chQuery := s.ClickHouseQuery
	var chLag time.Duration
	if chQuery != nil && s.CHIngestionLag != nil {
		chLag, _ = s.CHIngestionLag(ctx)
	}
	freshness := reports.PortfolioFreshness(to, chQuery != nil, chLag)

	byOwner := make(map[uuid.UUID]teamOwnerAccumulator)
	campaignIDs := make([]uuid.UUID, 0, len(campaignOwners))
	for campaignID, ownerID := range campaignOwners {
		if ownerID == uuid.Nil {
			continue
		}
		campaignIDs = append(campaignIDs, campaignID)
		if _, ok := byOwner[ownerID]; !ok {
			byOwner[ownerID] = teamOwnerAccumulator{}
		}
	}

	q := db.New(s.Pool)
	statRows, err := q.SumCustomerCampaignStatsInRange(ctx, db.SumCustomerCampaignStatsInRangeParams{
		CustomerID: domain.ToUUID(customerID),
		FromDate:   pgtype.Date{Time: from, Valid: true},
		ToDate:     pgtype.Date{Time: to, Valid: true},
	})
	if err != nil {
		return TeamMetricsResponse{}, err
	}
	for _, row := range statRows {
		campaignID := uuid.UUID(row.CampaignID.Bytes)
		ownerID, ok := campaignOwners[campaignID]
		if !ok || ownerID == uuid.Nil {
			continue
		}
		acc := byOwner[ownerID]
		acc.clicks += row.Clicks
		acc.conversions += row.Conversions
		byOwner[ownerID] = acc
	}

	if chQuery != nil && len(campaignIDs) > 0 {
		timeout := reports.ReportClickHouseQueryTimeout()
		if s.ReportCHTimeout != nil {
			timeout = s.ReportCHTimeout()
		}
		clickhouseCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		economicsByCampaign, econErr := reports.QueryCampaignEconomicsByCampaignCH(clickhouseCtx, chQuery, campaignIDs, from, to)
		if econErr == nil {
			for campaignID, ownerID := range campaignOwners {
				if ownerID == uuid.Nil {
					continue
				}
				econ, ok := economicsByCampaign[campaignID.String()]
				if !ok {
					continue
				}
				acc := byOwner[ownerID]
				acc.spendMicro += econ.SpendMicro
				acc.revenueMicro += econ.RevenueMicro
				byOwner[ownerID] = acc
			}
		}
	}

	var aggregateAcc teamOwnerAccumulator
	ownerRows := make([]TeamOwnerMetricsDTO, 0, len(byOwner))
	for ownerID, acc := range byOwner {
		aggregateAcc.clicks += acc.clicks
		aggregateAcc.conversions += acc.conversions
		aggregateAcc.spendMicro += acc.spendMicro
		aggregateAcc.revenueMicro += acc.revenueMicro
		ownerRows = append(ownerRows, TeamOwnerMetricsDTO{
			UserID: ownerID.String(),
			KPIs:   accumulatorToMetricsBlock(acc, freshness),
		})
	}
	sortTeamOwnerMetricsByROI(ownerRows)
	if len(ownerRows) > TeamMetricsByOwnerCap {
		ownerRows = ownerRows[:TeamMetricsByOwnerCap]
	}
	if err := attachTeamOwnerEmails(ctx, s.Pool, customerID, ownerRows); err != nil {
		return TeamMetricsResponse{}, err
	}

	resp := TeamMetricsResponse{
		CustomerID: customerID.String(),
		Period: TeamPeriodDTO{
			From: from.UTC().Format(time.RFC3339),
			To:   to.UTC().Format(time.RFC3339),
		},
		Aggregate: accumulatorToMetricsBlock(aggregateAcc, freshness),
		ByOwner:   ownerRows,
	}
	if resp.ByOwner == nil {
		resp.ByOwner = []TeamOwnerMetricsDTO{}
	}
	if snap, ok := authz.SnapshotFromContext(ctx); ok && snap.Mask == authz.MaskMasked {
		scrubTeamMetricsForMasked(&resp)
	}
	return resp, nil
}

func sortTeamOwnerMetricsByROI(rows []TeamOwnerMetricsDTO) {
	sort.SliceStable(rows, func(i, j int) bool {
		left := rows[i].KPIs.ROIPct
		right := rows[j].KPIs.ROIPct
		if left == right {
			return rows[i].KPIs.RevenueMicro > rows[j].KPIs.RevenueMicro
		}
		return left > right
	})
}

func scrubTeamMetricsForMasked(resp *TeamMetricsResponse) {
	if resp == nil {
		return
	}
	resp.Aggregate.RevenueMicro = 0
	resp.Aggregate.ProfitMicro = 0
	resp.Aggregate.ROIPct = 0
	resp.Aggregate.EPCMicro = 0
	for i := range resp.ByOwner {
		resp.ByOwner[i].KPIs.RevenueMicro = 0
		resp.ByOwner[i].KPIs.ProfitMicro = 0
		resp.ByOwner[i].KPIs.ROIPct = 0
		resp.ByOwner[i].KPIs.EPCMicro = 0
	}
}

func loadScopedCampaignOwners(ctx context.Context, pool *pgxpool.Pool, customerID uuid.UUID, scope teamscope.ListScope) (map[uuid.UUID]uuid.UUID, error) {
	out := make(map[uuid.UUID]uuid.UUID)
	if pool == nil || customerID == uuid.Nil {
		return out, nil
	}
	rows, err := pool.Query(ctx, `
		SELECT id, owner_user_id
		FROM campaigns
		WHERE customer_id = $1
		  AND deleted_at IS NULL
		  AND ($2::uuid IS NULL OR owner_user_id = $2)
		  AND (
		    $3::uuid[] IS NULL
		    OR cardinality($3::uuid[]) = 0
		    OR owner_user_id = ANY($3::uuid[])
		  )`,
		customerID,
		nullableUUID(scope.OwnerUserID),
		ownerUUIDSlice(scope.OwnerUserIDs),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var campaignID uuid.UUID
		var ownerID pgtype.UUID
		if err := rows.Scan(&campaignID, &ownerID); err != nil {
			return nil, err
		}
		if !ownerID.Valid {
			continue
		}
		out[campaignID] = uuid.UUID(ownerID.Bytes)
	}
	return out, rows.Err()
}

func nullableUUID(id pgtype.UUID) any {
	if !id.Valid {
		return nil
	}
	return uuid.UUID(id.Bytes)
}

func ownerUUIDSlice(ids []uuid.UUID) any {
	if len(ids) == 0 {
		return nil
	}
	return ids
}

func attachTeamOwnerEmails(ctx context.Context, pool *pgxpool.Pool, customerID uuid.UUID, rows []TeamOwnerMetricsDTO) error {
	if pool == nil || len(rows) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, 0, len(rows))
	for _, row := range rows {
		ownerID, err := uuid.Parse(row.UserID)
		if err != nil {
			continue
		}
		ids = append(ids, ownerID)
	}
	if len(ids) == 0 {
		return nil
	}
	emailRows, err := pool.Query(ctx, `
		SELECT id, email
		FROM users
		WHERE customer_id = $1 AND id = ANY($2::uuid[])`, customerID, ids)
	if err != nil {
		return err
	}
	defer emailRows.Close()
	emails := make(map[string]string, len(ids))
	for emailRows.Next() {
		var userID uuid.UUID
		var email string
		if err := emailRows.Scan(&userID, &email); err != nil {
			return err
		}
		emails[userID.String()] = email
	}
	if err := emailRows.Err(); err != nil {
		return err
	}
	for i := range rows {
		rows[i].Email = emails[rows[i].UserID]
	}
	return nil
}

func countTeamBudgetApprovals(ctx context.Context, pool *pgxpool.Pool, customerID uuid.UUID, actorUserID uuid.UUID) (pendingTotal int64, pendingForActor int64, err error) {
	if pool == nil || customerID == uuid.Nil {
		return 0, 0, nil
	}
	err = pool.QueryRow(ctx, `
		SELECT count(*)
		FROM team_budget_approvals
		WHERE customer_id = $1 AND status = 'PENDING'`, customerID).Scan(&pendingTotal)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, 0, nil
		}
		return 0, 0, err
	}
	if actorUserID == uuid.Nil {
		return pendingTotal, 0, nil
	}
	err = pool.QueryRow(ctx, `
		SELECT count(*)
		FROM team_budget_approvals
		WHERE customer_id = $1 AND user_id = $2 AND status = 'PENDING'`, customerID, actorUserID).Scan(&pendingForActor)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return pendingTotal, 0, nil
		}
		return 0, 0, err
	}
	return pendingTotal, pendingForActor, nil
}

func listTeamBudgetApprovalsForUser(ctx context.Context, pool *pgxpool.Pool, customerID, userID uuid.UUID, limit, offset int) ([]TeamBudgetApprovalDTO, int64, error) {
	if pool == nil {
		return nil, 0, errTeamServiceUnavailable()
	}
	limit = normalizeTeamBudgetApprovalsLimit(limit)
	offset = normalizeTeamBudgetApprovalsOffset(offset)

	var total int64
	if err := pool.QueryRow(ctx, `
		SELECT count(*)
		FROM team_budget_approvals
		WHERE customer_id = $1 AND user_id = $2`, customerID, userID).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := pool.Query(ctx, `
		SELECT id, user_id, campaign_id, requested_budget_micro, previous_budget_micro, status, created_at
		FROM team_budget_approvals
		WHERE customer_id = $1 AND user_id = $2
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4`, customerID, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := make([]TeamBudgetApprovalDTO, 0, limit)
	for rows.Next() {
		var row TeamBudgetApprovalDTO
		var id, rowUserID, campaignID uuid.UUID
		var created time.Time
		if err := rows.Scan(&id, &rowUserID, &campaignID, &row.RequestedBudgetMicro, &row.PreviousBudgetMicro, &row.Status, &created); err != nil {
			return nil, 0, err
		}
		row.ID = id.String()
		row.UserID = rowUserID.String()
		row.CampaignID = campaignID.String()
		row.CreatedAt = created.UTC().Format(time.RFC3339)
		row.CreatedAtDisplay = coldpath.RFC3339Display(row.CreatedAt)
		out = append(out, row)
	}
	return out, total, rows.Err()
}
