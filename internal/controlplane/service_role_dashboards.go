package controlplane

import (
	"context"
	"strings"
	"time"

	"espx/internal/controlplane/adminapi"
	"espx/internal/domain"
	"espx/internal/ledger"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *Service) compositeReads() *adminapi.CompositeReadService {
	cr := adminapi.NewCompositeReadService(s.GetPool(), s.cfg)
	if s.chQuery != nil {
		cr.SetCHQuery(s.chQuery)
	}
	return cr
}

func statementSpendMicro(lines []ledger.InvoiceLineDTO) int64 {
	var spend int64
	for _, line := range lines {
		if line.LedgerType == "FEE" && line.AmountMicro < 0 {
			spend += -line.AmountMicro
		}
	}
	return spend
}

func statementFeeMicro(lines []ledger.InvoiceLineDTO) int64 {
	var fee int64
	for _, line := range lines {
		if strings.Contains(line.LedgerType, "FEE") {
			if line.AmountMicro < 0 {
				fee += -line.AmountMicro
			} else {
				fee += line.AmountMicro
			}
		}
	}
	return fee
}

func sumAllDisputes(ctx context.Context, s *Service, customerID uuid.UUID) int64 {
	if s.payment == nil {
		return 0
	}
	const pageSize int32 = 50
	var offset int32
	var totalMicro int64
	for {
		disputes, err := s.ListDisputes(ctx, customerID.String(), pageSize, offset)
		if err != nil {
			return totalMicro
		}
		for _, d := range disputes.Disputes {
			totalMicro += d.AmountMicro
		}
		offset += int32(len(disputes.Disputes))
		if len(disputes.Disputes) < int(pageSize) || int64(offset) >= disputes.Total {
			break
		}
	}
	return totalMicro
}

type AdOpsDashboardDTO = adminapi.AdOpsDashboardDTO
type CFODashboardDTO = adminapi.CFODashboardDTO
type AccountantDashboardDTO = adminapi.AccountantDashboardDTO
type FraudDashboardDTO = adminapi.FraudDashboardDTO

// GetAdOpsDashboard returns unit economics and worst sources for a customer.
func (s *Service) GetAdOpsDashboard(ctx context.Context, customerID uuid.UUID) (AdOpsDashboardDTO, error) {
	portfolio, err := s.GetBuyerPortfolio(ctx, customerID)
	if err != nil {
		return AdOpsDashboardDTO{}, err
	}
	resp := AdOpsDashboardDTO{
		CustomerID: customerID.String(),
		Period:     portfolio.Period,
		Campaigns:  make([]adminapi.BuyerCampaignRowDTO, 0, len(portfolio.Campaigns)),
	}
	if portfolio.KPIs != nil {
		resp.KPIs = *portfolio.KPIs
	}
	for _, c := range portfolio.Campaigns {
		resp.Campaigns = append(resp.Campaigns, adminapi.BuyerCampaignRowDTO{
			ID:             c.ID,
			Name:           c.Name,
			Status:         c.Status,
			SpendMicro:     c.SpendMicro,
			BudgetMicro:    c.BudgetMicro,
			UtilizationPct: c.UtilizationPct,
			ROIPct:         0,
			PacingDriftPct: c.PacingDriftPct,
			OverspendRisk:  c.OverspendRisk,
		})
	}
	resp.WorstSources = s.worstIVTSources(ctx, customerID, portfolio.Period)
	return resp, nil
}

func (s *Service) worstIVTSources(ctx context.Context, customerID uuid.UUID, period adminapi.PeriodDTO) []adminapi.SourceRowDTO {
	if s.chQuery == nil {
		return []adminapi.SourceRowDTO{}
	}
	from, err := time.Parse(time.RFC3339, period.From)
	if err != nil {
		return nil
	}
	to, err := time.Parse(time.RFC3339, period.To)
	if err != nil {
		return nil
	}
	campaignIDs, err := adminapi.ListCustomerCampaignIDs(ctx, s.GetPool(), customerID)
	if err != nil || len(campaignIDs) == 0 {
		return nil
	}
	chCtx, cancel := context.WithTimeout(ctx, adminapi.ReportCHQueryTimeout())
	defer cancel()
	sources, err := adminapi.QueryWorstIVTSources(chCtx, s.chQuery, campaignIDs, from, to, 5)
	if err != nil || len(sources) == 0 {
		return nil
	}
	return sources
}

func worstSourcesFromCampaigns(campaigns []adminapi.BuyerCampaignRowDTO) []adminapi.SourceRowDTO {
	type scored struct {
		row   adminapi.SourceRowDTO
		score float64
	}
	scoredRows := make([]scored, 0, len(campaigns))
	for i := range campaigns {
		c := campaigns[i]
		if c.PacingDriftPct <= 0 && !c.OverspendRisk {
			continue
		}
		score := c.PacingDriftPct
		if c.OverspendRisk {
			score += 25
		}
		scoredRows = append(scoredRows, scored{
			row: adminapi.SourceRowDTO{
				CampaignID:   c.ID,
				Sub1:         c.Name,
				SpendMicro:   c.SpendMicro,
				IVTRate:      0,
				QualityScore: adminapi.CalcQualityFromDrift(c.PacingDriftPct),
			},
			score: score,
		})
	}
	for i := 0; i < len(scoredRows); i++ {
		for j := i + 1; j < len(scoredRows); j++ {
			if scoredRows[j].score > scoredRows[i].score {
				scoredRows[i], scoredRows[j] = scoredRows[j], scoredRows[i]
			}
		}
	}
	out := make([]adminapi.SourceRowDTO, 0, 5)
	for i := 0; i < len(scoredRows) && i < 5; i++ {
		out = append(out, scoredRows[i].row)
	}
	return out
}

// GetCFODashboard returns billed totals and dispute exposure for a customer.
func (s *Service) GetCFODashboard(ctx context.Context, customerID uuid.UUID) (CFODashboardDTO, error) {
	if customerID == uuid.Nil {
		return CFODashboardDTO{}, errValidation("customer_id is required")
	}
	now := time.Now().UTC()
	from := now.Add(-30 * 24 * time.Hour)

	reads := s.compositeReads()
	stmt, err := reads.BuildStatement(ctx, customerID, from, now)
	if err != nil {
		return CFODashboardDTO{}, err
	}
	spendMicro := statementSpendMicro(stmt.Lines)
	feeMicro := statementFeeMicro(stmt.Lines)
	if stmt.TaxBreakdown.TaxMicro > 0 {
		feeMicro += stmt.TaxBreakdown.TaxMicro
	}
	arAging := int64(0)
	if stmt.ClosingBalanceMicro < 0 {
		arAging = -stmt.ClosingBalanceMicro
	}

	return CFODashboardDTO{
		CustomerID: customerID.String(),
		Period: adminapi.PeriodDTO{
			From: from.Format(time.RFC3339),
			To:   now.Format(time.RFC3339),
		},
		BilledMicro:          stmt.Reconciliation.InvoiceTotalMicro,
		ARAgingMicro:         arAging,
		FeeTotalMicro:        feeMicro,
		DisputeExposureMicro: sumAllDisputes(ctx, s, customerID),
		KPIs: adminapi.MetricsBlockDTO{
			SpendMicro: spendMicro,
			Freshness: adminapi.DataFreshnessDTO{
				AsOf:        now.Format(time.RFC3339),
				Consistency: "strong",
				Stale:       s.chQuery == nil,
			},
		},
	}, nil
}

// GetAccountantDashboard returns close checklist and export job status.
func (s *Service) GetAccountantDashboard(ctx context.Context, customerID uuid.UUID) (AccountantDashboardDTO, error) {
	if customerID == uuid.Nil {
		return AccountantDashboardDTO{}, errValidation("customer_id is required")
	}
	now := time.Now().UTC()
	month := now.Format("2006-01")

	reads := s.compositeReads()
	inv, err := reads.GetInvariant(ctx, &customerID)
	if err != nil {
		return AccountantDashboardDTO{}, err
	}

	var countryCode, taxScheme string
	err = s.GetPool().QueryRow(ctx, `
		SELECT country_code, tax_scheme::text
		FROM billing.customer_tax_profiles WHERE customer_id = $1`,
		domain.ToUUID(customerID),
	).Scan(&countryCode, &taxScheme)
	if err != nil && err != pgx.ErrNoRows {
		return AccountantDashboardDTO{}, err
	}

	return AccountantDashboardDTO{
		CustomerID: customerID.String(),
		Period: adminapi.PeriodDTO{
			From: now.Add(-30 * 24 * time.Hour).Format(time.RFC3339),
			To:   now.Format(time.RFC3339),
		},
		Close: adminapi.AccountantCloseDTO{
			CustomerID:          customerID.String(),
			BillingMonth:        month,
			InvariantOK:         inv.OK,
			InvariantDeltaMicro: inv.DiffMicro,
		},
		TaxCountry: countryCode,
		TaxScheme:  taxScheme,
		ExportJobs: []adminapi.ExportJobStatusDTO{},
	}, nil
}

// GetFraudDashboard returns ML and IVT campaign signals for a customer.
func (s *Service) GetFraudDashboard(ctx context.Context, customerID uuid.UUID) (FraudDashboardDTO, error) {
	if customerID == uuid.Nil {
		return FraudDashboardDTO{}, errValidation("customer_id is required")
	}
	now := time.Now().UTC()
	var ghost int
	err := s.GetPool().QueryRow(ctx, `
		SELECT count(*)::int FROM campaigns
		WHERE customer_id = $1 AND ghost_ivt_enabled = TRUE AND deleted_at IS NULL`,
		domain.ToUUID(customerID),
	).Scan(&ghost)
	if err != nil {
		return FraudDashboardDTO{}, err
	}

	// ml_manual_labels is global ops queue; per-customer scoping needs campaign linkage.
	var labelsPending int

	edge, _ := FetchEdgeMetrics(ctx)
	return FraudDashboardDTO{
		CustomerID: customerID.String(),
		Period: adminapi.PeriodDTO{
			From: now.Add(-7 * 24 * time.Hour).Format(time.RFC3339),
			To:   now.Format(time.RFC3339),
		},
		GhostIVTCampaigns: ghost,
		LabelsPending:     labelsPending,
		EdgeBlockedFraud:  edge.Blocked["fraud_tier"],
	}, nil
}
