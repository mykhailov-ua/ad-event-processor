package dashboardadmin

import (
	"context"
	"time"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/reports"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type CampaignService struct {
	host CampaignHost
}

func NewCampaignService(host CampaignHost) *CampaignService {
	return &CampaignService{host: host}
}

func (st *CampaignService) GetCampaignDashboard(ctx context.Context, req CampaignDashboardRequest) (CampaignDashboardDTO, error) {
	if req.CampaignID == uuid.Nil {
		return CampaignDashboardDTO{}, st.host.ErrValidation("invalid campaign id")
	}

	from := req.From.UTC()
	to := req.To.UTC()
	if !to.After(from) {
		return CampaignDashboardDTO{}, st.host.ErrValidation("invalid time range")
	}

	q := db.New(st.host.Pool())
	camp, err := q.GetCampaign(ctx, domain.ToUUID(req.CampaignID))
	if err != nil {
		return CampaignDashboardDTO{}, st.host.MapCampaignNotFound(err)
	}
	customerID := uuid.UUID(camp.CustomerID.Bytes)

	var spendMicro, revenueMicro, conversions int64
	chAvailable := st.host.ClickHouseQuery() != nil
	usedCHMoney := false

	if chAvailable {
		clickhouseCtx, cancel := context.WithTimeout(ctx, st.host.ReportCHTimeout())
		defer cancel()
		econ, err := reports.QueryCampaignEconomicsCH(clickhouseCtx, st.host.ClickHouseQuery(), req.CampaignID, from, to)
		if err != nil {
			return CampaignDashboardDTO{}, err
		}
		spendMicro = econ.SpendMicro
		revenueMicro = econ.RevenueMicro
		conversions = econ.Conversions
		usedCHMoney = spendMicro > 0 || revenueMicro > 0
	}

	if spendMicro == 0 && revenueMicro == 0 {
		spendMicro = camp.CurrentSpend
		stats, err := q.SumCampaignStatsInRange(ctx, db.SumCampaignStatsInRangeParams{
			CampaignID: domain.ToUUID(req.CampaignID),
			FromDate:   pgtype.Date{Time: from, Valid: true},
			ToDate:     pgtype.Date{Time: to, Valid: true},
		})
		if err != nil {
			return CampaignDashboardDTO{}, err
		}
		if conversions == 0 {
			conversions = stats.Conversions
		}
	}

	profitMicro := revenueMicro - spendMicro
	var cpaMicro int64
	if conversions > 0 && spendMicro > 0 {
		cpaMicro = spendMicro / conversions
	}

	now := time.Now().UTC()
	chLag := st.host.ClickHouseLag(ctx)
	freshness := reports.CampaignDashboardFreshness(now, usedCHMoney, chLag, chAvailable)

	var series []DashboardSeriesPointDTO
	if st.host.ClickHouseQuery() != nil {
		clickhouseCtx, cancel := context.WithTimeout(ctx, st.host.ReportCHTimeout())
		series, _ = reports.QueryCustomerDashboardSeries(clickhouseCtx, st.host.Pool(), st.host.ClickHouseQuery(), customerID, []uuid.UUID{req.CampaignID}, from, to, reports.ChartGranularityDay)
		cancel()
	}

	breakdown, err := st.queryCampaignReportBreakdown(ctx, req.CampaignID, from, to, req.Dimension)
	if err != nil {
		return CampaignDashboardDTO{}, err
	}
	breakdown = ApplyCampaignReportBreakdownQuery(breakdown, req.Q, req.SortField, req.SortDesc)

	return CampaignDashboardDTO{
		CampaignID:   req.CampaignID.String(),
		CampaignName: camp.Name,
		Period: PeriodDTO{
			From: from.Format(time.RFC3339),
			To:   to.Format(time.RFC3339),
		},
		ReportDimension: string(req.Dimension),
		Series:          series,
		Breakdown:       breakdown,
		KPIs: MetricsBlockDTO{
			SpendMicro:   spendMicro,
			RevenueMicro: revenueMicro,
			ProfitMicro:  profitMicro,
			ROIPct:       reports.CalcROIPct(profitMicro, spendMicro),
			Conversions:  conversions,
			CPAMicro:     cpaMicro,
			Freshness:    freshness,
		},
		Freshness: freshness,
	}, nil
}
