package controlplane

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/bidshard/ad-event-processor/internal/ledger"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *Service) compositeReads() *CompositeReadService {
	cr := NewCompositeReadService(s.GetPool(), s.cfg)
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

func (s *Service) GetAdOpsDashboard(ctx context.Context, customerID uuid.UUID) (AdOpsDashboardDTO, error) {
	portfolio, err := s.GetBuyerPortfolio(ctx, customerID)
	if err != nil {
		return AdOpsDashboardDTO{}, err
	}
	resp := AdOpsDashboardDTO{
		CustomerID: customerID.String(),
		Period:     portfolio.Period,
		Campaigns:  make([]BuyerCampaignRowDTO, 0, len(portfolio.Campaigns)),
	}
	if portfolio.KPIs != nil {
		resp.KPIs = *portfolio.KPIs
	}
	for _, c := range portfolio.Campaigns {
		resp.Campaigns = append(resp.Campaigns, BuyerCampaignRowDTO{
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

func (s *Service) worstIVTSources(ctx context.Context, customerID uuid.UUID, period PeriodDTO) []SourceRowDTO {
	if s.chQuery == nil {
		return []SourceRowDTO{}
	}
	from, err := time.Parse(time.RFC3339, period.From)
	if err != nil {
		return nil
	}
	to, err := time.Parse(time.RFC3339, period.To)
	if err != nil {
		return nil
	}
	campaignIDs, err := ListCustomerCampaignIDs(ctx, s.GetPool(), customerID)
	if err != nil || len(campaignIDs) == 0 {
		return nil
	}
	chCtx, cancel := context.WithTimeout(ctx, ReportCHQueryTimeout())
	defer cancel()
	sources, err := QueryWorstIVTSources(chCtx, s.chQuery, campaignIDs, from, to, 5)
	if err != nil || len(sources) == 0 {
		return nil
	}
	return sources
}

func worstSourcesFromCampaigns(campaigns []BuyerCampaignRowDTO) []SourceRowDTO {
	type scored struct {
		row   SourceRowDTO
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
			row: SourceRowDTO{
				CampaignID:   c.ID,
				Sub1:         c.Name,
				SpendMicro:   c.SpendMicro,
				IVTRate:      0,
				QualityScore: CalcQualityFromDrift(c.PacingDriftPct),
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
	out := make([]SourceRowDTO, 0, 5)
	for i := 0; i < len(scoredRows) && i < 5; i++ {
		out = append(out, scoredRows[i].row)
	}
	return out
}

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
		Period: PeriodDTO{
			From: from.Format(time.RFC3339),
			To:   now.Format(time.RFC3339),
		},
		BilledMicro:          stmt.Reconciliation.InvoiceTotalMicro,
		ARAgingMicro:         arAging,
		FeeTotalMicro:        feeMicro,
		DisputeExposureMicro: sumAllDisputes(ctx, s, customerID),
		KPIs: MetricsBlockDTO{
			SpendMicro: spendMicro,
			Freshness: DataFreshnessDTO{
				AsOf:        now.Format(time.RFC3339),
				Consistency: "strong",
				Stale:       s.chQuery == nil,
			},
		},
	}, nil
}

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
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return AccountantDashboardDTO{}, err
	}

	return AccountantDashboardDTO{
		CustomerID: customerID.String(),
		Period: PeriodDTO{
			From: now.Add(-30 * 24 * time.Hour).Format(time.RFC3339),
			To:   now.Format(time.RFC3339),
		},
		Close: AccountantCloseDTO{
			CustomerID:          customerID.String(),
			BillingMonth:        month,
			InvariantOK:         inv.OK,
			InvariantDeltaMicro: inv.DiffMicro,
		},
		TaxCountry: countryCode,
		TaxScheme:  taxScheme,
		ExportJobs: []ExportJobStatusDTO{},
	}, nil
}

const fraudTierThresholdScopePlatformDefault = "platform_default"

func (s *Service) GetFraudDashboard(ctx context.Context, customerID uuid.UUID) (FraudDashboardDTO, error) {
	if customerID == uuid.Nil {
		return FraudDashboardDTO{}, errValidation("customer_id is required")
	}
	now := time.Now().UTC()
	from := now.Add(-7 * 24 * time.Hour)
	var ghost int
	err := s.GetPool().QueryRow(ctx, `
		SELECT count(*)::int FROM campaigns
		WHERE customer_id = $1 AND ghost_ivt_enabled = TRUE AND deleted_at IS NULL`,
		domain.ToUUID(customerID),
	).Scan(&ghost)
	if err != nil {
		return FraudDashboardDTO{}, err
	}

	var labelsPending int
	if err := s.GetPool().QueryRow(ctx, `
		SELECT count(*)::int FROM ml_manual_labels
		WHERE customer_id = $1 AND created_at >= $2`, domain.ToUUID(customerID), from).Scan(&labelsPending); err != nil {
		return FraudDashboardDTO{}, fmt.Errorf("count pending ml labels: %w", err)
	}

	mlSnap, err := s.fraudMLSnapshot(ctx)
	if err != nil {
		return FraudDashboardDTO{}, err
	}

	recentLabels, err := s.listRecentMLLabelsForCustomer(ctx, customerID, 5)
	if err != nil {
		return FraudDashboardDTO{}, fmt.Errorf("list recent ml labels: %w", err)
	}

	geoHints := s.fraudGeoHints(ctx, customerID, from, now)

	edge, err := FetchEdgeMetrics(ctx)
	if err != nil {
		return FraudDashboardDTO{}, fmt.Errorf("fetch edge metrics: %w", err)
	}
	return FraudDashboardDTO{
		CustomerID: customerID.String(),
		Period: PeriodDTO{
			From: from.Format(time.RFC3339),
			To:   now.Format(time.RFC3339),
		},
		GhostIVTCampaigns:  ghost,
		LabelsPending:      labelsPending,
		EdgeBlockedFraud:   edge.Blocked["fraud_tier"],
		MLActiveVersionID:  mlSnap.VersionID,
		MLArtifactHash:     mlSnap.ArtifactHash,
		MLPrecision:        mlSnap.Precision,
		MLRecall:           mlSnap.Recall,
		MLDriftDetected:    mlSnap.DriftDetected,
		MLDriftSummary:     mlSnap.DriftSummary,
		MLEvalGeneratedAt:  mlSnap.EvalGeneratedAt,
		MLEvalStatus:       mlSnap.EvalStatus,
		MLEvalStale:        mlSnap.EvalStale,
		MLLabelMethod:      mlSnap.LabelMethod,
		MLShardsConsistent: mlSnap.ShardsConsistent,
		FraudTierThresholds: FraudTierThresholdsDTO{
			Scope:      fraudTierThresholdScopePlatformDefault,
			PassMax:    int(domain.DefaultFraudThresholdPass),
			SuspectMax: int(domain.DefaultFraudThresholdSuspect),
			IVTMax:     int(domain.DefaultFraudThresholdIVT),
			BlockAbove: int(domain.DefaultFraudThresholdBlock),
		},
		GeoHints:     geoHints,
		RecentLabels: recentLabels,
	}, nil
}

func (s *Service) listRecentMLLabelsForCustomer(ctx context.Context, customerID uuid.UUID, limit int) ([]MLManualLabelDTO, error) {
	return s.ListMLManualLabelsForCustomer(ctx, customerID, limit)
}

func (s *Service) fraudGeoHints(ctx context.Context, customerID uuid.UUID, from, to time.Time) []FraudGeoHintDTO {
	if s.chQuery == nil {
		return nil
	}
	campaignIDs, err := ListCustomerCampaignIDs(ctx, s.GetPool(), customerID)
	if err != nil || len(campaignIDs) == 0 {
		return nil
	}
	chCtx, cancel := context.WithTimeout(ctx, ReportCHQueryTimeout())
	defer cancel()
	hints, err := QueryWorstIVTCountries(chCtx, s.chQuery, campaignIDs, from, to, 5)
	if err != nil {
		return nil
	}
	return hints
}
