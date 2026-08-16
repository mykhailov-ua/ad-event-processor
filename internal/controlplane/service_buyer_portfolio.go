package controlplane

import (
	"context"
	"time"

	"github.com/bidshard/ad-event-processor/internal/controlplane/adminapi"
	"github.com/bidshard/ad-event-processor/internal/domain"
	db "github.com/bidshard/ad-event-processor/internal/domain/db"
	"github.com/bidshard/ad-event-processor/pkg/money"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type BuyerPortfolioDTO = adminapi.BuyerPortfolioDTO
type BuyerAttentionDTO = adminapi.BuyerAttentionDTO
type BuyerCampaignPortfolioRowDTO = adminapi.BuyerCampaignPortfolioRowDTO

// GetBuyerPortfolio returns buyer portfolio counters and per-campaign 7d stats.
func (s *Service) GetBuyerPortfolio(ctx context.Context, customerID uuid.UUID) (BuyerPortfolioDTO, error) {
	if customerID == uuid.Nil {
		return BuyerPortfolioDTO{}, errValidation("customer_id is required")
	}
	now := time.Now().UTC()
	from := now.Add(-7 * 24 * time.Hour)
	to := now

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

	resp := BuyerPortfolioDTO{
		CustomerID: customerID.String(),
		Period: adminapi.PeriodDTO{
			From: from.Format(time.RFC3339),
			To:   to.Format(time.RFC3339),
		},
		Attention: make([]BuyerAttentionDTO, 0, 4),
		Campaigns: make([]BuyerCampaignPortfolioRowDTO, 0, len(campaigns)),
		KPIs: &adminapi.MetricsBlockDTO{
			Freshness: adminapi.DataFreshnessDTO{
				AsOf:        to.Format(time.RFC3339),
				Consistency: "strong",
				Stale:       s.chQuery == nil,
			},
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
	marginBreaches, _ := s.batchCampaignMarginBreach(ctx, activeCampaignIDs)
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

		spendMicro, _ := money.ParseDecimal(c.CurrentSpend)
		budgetMicro, _ := money.ParseDecimal(c.BudgetLimit)
		totalSpendMicro += spendMicro
		util := campaignUtilizationPct(spendMicro, budgetMicro)
		drift := pacingDriftPct(st.Impressions, c.Status)
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
			ID:             c.ID,
			Name:           c.Name,
			Status:         c.Status,
			PacingMode:     c.PacingMode,
			Impressions7d:  st.Impressions,
			Clicks7d:       st.Clicks,
			SpendMicro:     spendMicro,
			BudgetMicro:    budgetMicro,
			UtilizationPct: util,
			PacingDriftPct: drift,
			OverspendRisk:  risk,
			MarginBreach:   marginBreach,
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
