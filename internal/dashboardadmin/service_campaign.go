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

func (st *CampaignService) GetCampaignDashboard(ctx context.Context, campaignID uuid.UUID) (CampaignDashboardDTO, error) {
	if campaignID == uuid.Nil {
		return CampaignDashboardDTO{}, st.host.ErrValidation("invalid campaign id")
	}

	now := time.Now().UTC()
	from := now.Add(-30 * 24 * time.Hour)

	q := db.New(st.host.Pool())
	camp, err := q.GetCampaign(ctx, domain.ToUUID(campaignID))
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
		econ, err := reports.QueryCampaignEconomicsCH(clickhouseCtx, st.host.ClickHouseQuery(), campaignID, from, now)
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
			CampaignID: domain.ToUUID(campaignID),
			FromDate:   pgtype.Date{Time: from.UTC(), Valid: true},
			ToDate:     pgtype.Date{Time: now.UTC(), Valid: true},
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

	chLag := st.host.ClickHouseLag(ctx)
	freshness := reports.CampaignDashboardFreshness(now, usedCHMoney, chLag, chAvailable)

	var series []DashboardSeriesPointDTO
	if st.host.ClickHouseQuery() != nil {
		clickhouseCtx, cancel := context.WithTimeout(ctx, st.host.ReportCHTimeout())
		series, _ = reports.QueryCustomerDashboardSeries(clickhouseCtx, st.host.Pool(), st.host.ClickHouseQuery(), customerID, []uuid.UUID{campaignID}, from, now)
		cancel()
	}

	return CampaignDashboardDTO{
		CampaignID: campaignID.String(),
		Series:     series,
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
