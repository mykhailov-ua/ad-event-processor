package dashboardadmin

import (
	"context"
	"fmt"
	"time"

	"ad-event-processor/internal/campaign"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/reports"
	"ad-event-processor/pkg/money"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

const portfolioCampaignPageSize int32 = 100

func (p *Portfolio) GetBuyerPortfolio(ctx context.Context, customerID uuid.UUID) (BuyerPortfolioDTO, error) {
	now := time.Now().UTC()
	return p.GetBuyerPortfolioRange(ctx, customerID, nil, now.Add(-7*24*time.Hour), now, reports.ChartGranularityDay)
}

func (p *Portfolio) GetBuyerPortfolioRange(
	ctx context.Context,
	customerID uuid.UUID,
	campaignFilter *uuid.UUID,
	from, to time.Time,
	seriesGranularity reports.ChartGranularity,
) (BuyerPortfolioDTO, error) {
	if p == nil || p.host == nil {
		return BuyerPortfolioDTO{}, fmt.Errorf("portfolio service unavailable")
	}
	if customerID == uuid.Nil {
		return BuyerPortfolioDTO{}, p.host.ErrValidation("customer_id is required")
	}
	now := time.Now().UTC()
	if from.IsZero() {
		from = now.Add(-7 * 24 * time.Hour)
	}
	if to.IsZero() {
		to = now
	}
	if err := reports.ValidateChartRange(from, to); err != nil {
		return BuyerPortfolioDTO{}, err
	}

	campaigns, err := p.listAllCampaigns(ctx, customerID, "")
	if err != nil {
		return BuyerPortfolioDTO{}, err
	}
	if campaignFilter != nil && *campaignFilter != uuid.Nil {
		filtered := make([]campaign.CampaignDTO, 0, 1)
		for _, c := range campaigns {
			if c.ID == campaignFilter.String() {
				filtered = append(filtered, c)
				break
			}
		}
		campaigns = filtered
	}

	q := db.New(p.host.Pool())
	statRows, err := q.SumCustomerCampaignStatsInRange(ctx, db.SumCustomerCampaignStatsInRangeParams{
		CustomerID: domain.ToUUID(customerID),
		FromDate:   pgtype.Date{Time: from, Valid: true},
		ToDate:     pgtype.Date{Time: to, Valid: true},
	})
	if err != nil {
		return BuyerPortfolioDTO{}, err
	}
	statsByCampaign := make(map[string]db.SumCustomerCampaignStatsInRangeRow, len(statRows))
	for _, row := range statRows {
		statsByCampaign[uuid.UUID(row.CampaignID.Bytes).String()] = row
	}

	chQuery := p.host.ClickHouseQuery()
	var chLag time.Duration
	if chQuery != nil {
		chLag, _ = p.host.ClickHouseIngestionLag(ctx)
	}
	resp := BuyerPortfolioDTO{
		CustomerID: customerID.String(),
		Period: PeriodDTO{
			From: from.Format(time.RFC3339),
			To:   to.Format(time.RFC3339),
		},
		Attention: make([]BuyerAttentionDTO, 0, 4),
		Campaigns: make([]BuyerCampaignPortfolioRowDTO, 0, len(campaigns)),
		KPIs: &MetricsBlockDTO{
			Freshness: reports.PortfolioFreshness(to, chQuery != nil, chLag),
		},
	}

	var totalSpendMicro int64
	var totalConversions int64

	activeCampaignIDs := make([]uuid.UUID, 0, len(campaigns))
	for _, c := range campaigns {
		if c.Status == "ACTIVE" {
			if campID, parseErr := uuid.Parse(c.ID); parseErr == nil {
				activeCampaignIDs = append(activeCampaignIDs, campID)
			}
		}
	}
	marginBreaches, err := p.host.BatchCampaignMarginBreach(ctx, activeCampaignIDs)
	if err != nil {
		return BuyerPortfolioDTO{}, fmt.Errorf("batch campaign margin breach: %w", err)
	}
	marginBreachByID := make(map[string]bool, len(activeCampaignIDs))
	for _, id := range activeCampaignIDs {
		marginBreachByID[id.String()] = marginBreaches[id]
	}

	for _, c := range campaigns {
		switch c.Status {
		case "ACTIVE":
			resp.Active++
		case "PAUSED":
			resp.Paused++
			resp.Attention = append(resp.Attention, BuyerAttentionDTO{
				ID: c.ID, Name: c.Name, Reason: "Campaign is paused",
			})
		case "ARCHIVED":
			resp.Archived++
		}
		if c.Status == "ACTIVE" && c.PacingMode != "" && c.PacingMode != "even" {
			resp.Attention = append(resp.Attention, BuyerAttentionDTO{
				ID: c.ID, Name: c.Name, Reason: "Pacing mode: " + c.PacingMode,
			})
		}
		st := statsByCampaign[c.ID]
		resp.Impressions7d += st.Impressions
		resp.Clicks7d += st.Clicks
		totalConversions += st.Conversions

		spendMicro, err := money.ParseDecimal(c.CurrentSpend)
		if err != nil {
			return BuyerPortfolioDTO{}, fmt.Errorf("campaign %s current_spend: %w", c.ID, err)
		}
		budgetMicro, err := money.ParseDecimal(c.BudgetLimit)
		if err != nil {
			return BuyerPortfolioDTO{}, fmt.Errorf("campaign %s budget_limit: %w", c.ID, err)
		}
		totalSpendMicro += spendMicro
		util := campaignUtilizationPct(spendMicro, budgetMicro)
		estimatedDrift := pacingDriftPct(st.Impressions, c.Status)
		drift := estimatedDrift
		risk := overspendRisk(util, c.PacingMode, c.Status)
		if risk {
			resp.OverspendCount++
			resp.Attention = append(resp.Attention, BuyerAttentionDTO{
				ID: c.ID, Name: c.Name, Reason: "Overspend risk",
			})
		}

		marginBreach := false
		if c.Status == "ACTIVE" {
			marginBreach = marginBreachByID[c.ID]
		}

		resp.Campaigns = append(resp.Campaigns, BuyerCampaignPortfolioRowDTO{
			ID:                      c.ID,
			Name:                    c.Name,
			Status:                  c.Status,
			PacingMode:              c.PacingMode,
			Impressions7d:           st.Impressions,
			Clicks7d:                st.Clicks,
			SpendMicro:              spendMicro,
			BudgetMicro:             budgetMicro,
			UtilizationPct:          util,
			PacingDriftPct:          drift,
			EstimatedPacingDriftPct: estimatedDrift,
			OverspendRisk:           risk,
			MarginBreach:            marginBreach,
		})
	}
	if resp.KPIs != nil {
		resp.KPIs.SpendMicro = totalSpendMicro
		resp.KPIs.Conversions = totalConversions
		if totalConversions > 0 && totalSpendMicro > 0 {
			resp.KPIs.CPAMicro = totalSpendMicro / totalConversions
		}
	}
	if len(resp.Attention) > 8 {
		resp.Attention = resp.Attention[:8]
	}
	enrichBuyerPortfolioCommercial(&resp)
	scopedCampaignIDs := activeCampaignIDs
	if campaignFilter != nil && *campaignFilter != uuid.Nil {
		scopedCampaignIDs = []uuid.UUID{*campaignFilter}
	}
	if series, seriesErr := reports.QueryCustomerDashboardSeries(ctx, p.host.Pool(), chQuery, customerID, scopedCampaignIDs, from, to, seriesGranularity); seriesErr == nil {
		resp.Series = series
		applySeriesToKPIs(resp.KPIs, series)
	}
	breakdownInputs := make([]campaignBreakdownInput, 0, len(campaigns))
	for _, c := range campaigns {
		st := statsByCampaign[c.ID]
		breakdownInputs = append(breakdownInputs, campaignBreakdownInput{
			id:          c.ID,
			name:        c.Name,
			clicks:      st.Clicks,
			conversions: st.Conversions,
		})
	}
	resp.Breakdowns = &DashboardBreakdownsDTO{
		Campaigns: buildCampaignBreakdownTable(breakdownInputs),
		Sources:   reports.DashboardBreakdownTableDTO{Rows: []reports.DashboardBreakdownRowDTO{}},
		Landers:   reports.DashboardBreakdownTableDTO{Rows: []reports.DashboardBreakdownRowDTO{}},
		Offers:    reports.DashboardBreakdownTableDTO{Rows: []reports.DashboardBreakdownRowDTO{}},
	}
	if chQuery != nil && len(scopedCampaignIDs) > 0 {
		clickhouseCtx, cancel := context.WithTimeout(ctx, p.host.ReportCHTimeout())
		defer cancel()
		if uniqueClicks, ucErr := reports.QueryCustomerUniqueClicksCH(clickhouseCtx, chQuery, scopedCampaignIDs, from, to); ucErr == nil {
			resp.UniqueClicks7d = uniqueClicks
			if resp.KPIs != nil {
				resp.KPIs.UniqueClicks = uniqueClicks
			}
		}
		uniqueByCampaign, _ := reports.QueryUniqueClicksByCampaignCH(clickhouseCtx, chQuery, scopedCampaignIDs, from, to)
		economicsByCampaign, _ := reports.QueryCampaignEconomicsByCampaignCH(clickhouseCtx, chQuery, scopedCampaignIDs, from, to)
		for i := range breakdownInputs {
			input := &breakdownInputs[i]
			input.uniqueClicks = uniqueByCampaign[input.id]
			if econ, ok := economicsByCampaign[input.id]; ok {
				input.costMicro = econ.SpendMicro
				input.revenueMicro = econ.RevenueMicro
			}
		}
		resp.Breakdowns.Campaigns = buildCampaignBreakdownTable(breakdownInputs)
		totalEvents, blocked, silent, err := reports.QueryCustomerFraudOverview(clickhouseCtx, chQuery, scopedCampaignIDs, from, to)
		if err == nil {
			freshness := reports.PortfolioFreshness(to, true, chLag)
			overview := reports.BuildCustomerFraudOverview(totalEvents, blocked, silent, freshness)
			reports.AttachInvalidSpendKPI(&overview, blocked, silent, totalEvents, 0, reports.ComputeAttributionCoverage(totalEvents, totalEvents))
			if fraudSeries, seriesErr := reports.QueryCustomerFraudDailySeries(clickhouseCtx, chQuery, scopedCampaignIDs, from, to); seriesErr == nil {
				overview.Series = fraudSeries
			}
			resp.Fraud = &overview
		}
		if sources, sourcesErr := reports.QuerySourceBreakdownCH(clickhouseCtx, chQuery, scopedCampaignIDs, from, to, buyerBreakdownTopN); sourcesErr == nil {
			resp.Breakdowns.Sources = sources
		}
		if recent, recentErr := reports.QueryRecentClickLogCH(clickhouseCtx, chQuery, scopedCampaignIDs, from, to, 20); recentErr == nil {
			resp.RecentClicks = recent
		}
	}
	if resp.KPIs != nil && resp.KPIs.CostMicro == 0 && resp.KPIs.SpendMicro > 0 {
		resp.KPIs.CostMicro = resp.KPIs.SpendMicro
	}
	finalizeBuyerPortfolioKPIs(resp.KPIs, resp.Clicks7d)
	return resp, nil
}

func finalizeBuyerPortfolioKPIs(kpis *MetricsBlockDTO, clicks int64) {
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

func applySeriesToKPIs(kpis *MetricsBlockDTO, series []reports.DashboardSeriesPointDTO) {
	if kpis == nil || len(series) == 0 {
		return
	}
	var costMicro, revenueMicro, profitMicro int64
	for _, point := range series {
		spend := point.SpendMicro
		if spend == 0 {
			spend = point.SpendMicros
		}
		costMicro += spend
		revenueMicro += point.RevenueMicro
		profitMicro += point.ProfitMicro
	}
	if revenueMicro > 0 {
		kpis.RevenueMicro = revenueMicro
		kpis.ProfitMicro = profitMicro
	}
	if costMicro > 0 {
		kpis.CostMicro = costMicro
		kpis.SpendMicro = costMicro
	}
}

func (p *Portfolio) listAllCampaigns(ctx context.Context, customerID uuid.UUID, status string) ([]campaign.CampaignDTO, error) {
	var all []campaign.CampaignDTO
	var offset int32
	for {
		batch, total, err := p.host.ListCampaigns(ctx, customerID, status, portfolioCampaignPageSize, offset)
		if err != nil {
			return nil, err
		}
		all = append(all, batch...)
		offset += int32(len(batch))
		if len(batch) == 0 || int64(offset) >= total {
			break
		}
	}
	return all, nil
}
