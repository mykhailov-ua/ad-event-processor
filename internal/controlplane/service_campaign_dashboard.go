package controlplane

import (
	"context"
	"time"

	"github.com/bidshard/ad-event-processor/internal/domain"
	db "github.com/bidshard/ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func (s *Service) GetCampaignDashboard(ctx context.Context, campaignID uuid.UUID) (CampaignDashboardDTO, error) {
	if campaignID == uuid.Nil {
		return CampaignDashboardDTO{}, errValidation("invalid campaign id")
	}

	now := time.Now().UTC()
	from := now.Add(-30 * 24 * time.Hour)

	var spendMicro, revenueMicro, conversions int64
	stale := s.chQuery == nil

	if s.chQuery != nil {
		chCtx, cancel := context.WithTimeout(ctx, ReportCHQueryTimeout())
		defer cancel()
		econ, err := QueryCampaignEconomicsCH(chCtx, s.chQuery, campaignID, from, now)
		if err != nil {
			return CampaignDashboardDTO{}, err
		}
		spendMicro = econ.SpendMicro
		revenueMicro = econ.RevenueMicro
		conversions = econ.Conversions
	}

	if spendMicro == 0 && revenueMicro == 0 {
		q := db.New(s.GetPool())
		camp, err := q.GetCampaign(ctx, domain.ToUUID(campaignID))
		if err != nil {
			return CampaignDashboardDTO{}, mapNotFound(err, ErrCampaignNotFound)
		}
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

	freshness := DataFreshnessDTO{
		AsOf:        now.Format(time.RFC3339),
		Consistency: "eventual",
		Stale:       stale,
	}
	if !stale {
		freshness.Consistency = "strong"
	}

	return CampaignDashboardDTO{
		CampaignID: campaignID.String(),
		KPIs: MetricsBlockDTO{
			SpendMicro:   spendMicro,
			RevenueMicro: revenueMicro,
			ProfitMicro:  profitMicro,
			ROIPct:       CalcROIPct(profitMicro, spendMicro),
			Conversions:  conversions,
			CPAMicro:     cpaMicro,
			Freshness:    freshness,
		},
		Freshness: freshness,
	}, nil
}
