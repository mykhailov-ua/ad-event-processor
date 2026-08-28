package controlplane

import (
	"context"
	"fmt"
	"time"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/pkg/money"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func (s *Service) GetBuyerPortfolio(ctx context.Context, customerID uuid.UUID) (BuyerPortfolioDTO, error) {
	now := time.Now().UTC()
	return s.GetBuyerPortfolioRange(ctx, customerID, now.Add(-7*24*time.Hour), now)
}

func (s *Service) GetBuyerPortfolioRange(ctx context.Context, customerID uuid.UUID, from, to time.Time) (BuyerPortfolioDTO, error) {
	if customerID == uuid.Nil {
		return BuyerPortfolioDTO{}, errValidation("customer_id is required")
	}
	now := time.Now().UTC()
	if from.IsZero() {
		from = now.Add(-7 * 24 * time.Hour)
	}
	if to.IsZero() {
		to = now
	}
	if err := validateChartRange(from, to); err != nil {
		return BuyerPortfolioDTO{}, err
	}

	campaigns, err := s.listAllCampaigns(ctx, customerID, "")
	if err != nil {
		return BuyerPortfolioDTO{}, err
	}

	q := db.New(s.GetPool())
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

	var chLag time.Duration
	if s.clickhouseQuery != nil {
		chLag, _ = s.clickHouseIngestionLag(ctx)
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
			Freshness: portfolioFreshness(to, s.clickhouseQuery != nil, chLag),
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
	marginBreaches, err := s.batchCampaignMarginBreach(ctx, activeCampaignIDs)
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
	if s.clickhouseQuery != nil && len(activeCampaignIDs) > 0 {
		clickhouseCtx, cancel := context.WithTimeout(ctx, reportClickHouseQueryTimeout)
		defer cancel()
		totalEvents, blocked, silent, err := queryCustomerFraudOverview(clickhouseCtx, s.clickhouseQuery, activeCampaignIDs, from, to)
		if err == nil {
			overview := buildCustomerFraudOverview(totalEvents, blocked, silent, portfolioFreshness(to, true, chLag))
			attachInvalidSpendKPI(&overview, blocked, silent, totalEvents, 0, computeAttributionCoverage(totalEvents, totalEvents))
			if series, seriesErr := queryCustomerFraudDailySeries(clickhouseCtx, s.clickhouseQuery, activeCampaignIDs, from, to); seriesErr == nil {
				overview.Series = series
			}
			resp.Fraud = &overview
		}
	}
	if series, seriesErr := queryCustomerDashboardSeries(ctx, s.GetPool(), s.clickhouseQuery, customerID, activeCampaignIDs, from, to); seriesErr == nil {
		resp.Series = series
	}
	return resp, nil
}

const portfolioCampaignPageSize int32 = 100

func (s *Service) listAllCampaigns(ctx context.Context, customerID uuid.UUID, status string) ([]CampaignDTO, error) {
	var all []CampaignDTO
	var offset int32
	for {
		batch, total, err := s.ListCampaigns(ctx, customerID, status, portfolioCampaignPageSize, offset)
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
