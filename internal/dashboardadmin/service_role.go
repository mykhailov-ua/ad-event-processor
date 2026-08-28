package dashboardadmin

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/ledger"
	"ad-event-processor/internal/reports"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type RoleService struct {
	host       RoleHost
	reportHost RoleReportHost
}

func NewRoleService(host RoleHost, reportHost RoleReportHost) *RoleService {
	return &RoleService{host: host, reportHost: reportHost}
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

func (st *RoleService) GetAdOpsDashboard(ctx context.Context, customerID uuid.UUID) (AdOpsDashboardDTO, error) {
	portfolio, err := st.host.GetBuyerPortfolio(ctx, customerID)
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
			PacingDriftPct: c.PacingDriftPct,
			OverspendRisk:  c.OverspendRisk,
		})
	}
	resp.WorstSources = st.worstIVTSources(ctx, customerID, portfolio.Period)
	return resp, nil
}

func (st *RoleService) worstIVTSources(ctx context.Context, customerID uuid.UUID, period PeriodDTO) []SourceRowDTO {
	if st.host.ClickHouseQuery() == nil || st.reportHost == nil {
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
	campaignIDs, err := st.reportHost.ListCustomerCampaignIDs(ctx, customerID)
	if err != nil || len(campaignIDs) == 0 {
		return nil
	}
	clickhouseCtx, cancel := context.WithTimeout(ctx, st.host.ReportCHTimeout())
	defer cancel()
	sources, err := st.reportHost.QueryWorstIVTSources(clickhouseCtx, campaignIDs, from, to, 5)
	if err != nil || len(sources) == 0 {
		return nil
	}
	return sourceRowsFromReports(sources)
}

func sourceRowsFromReports(rows []reports.SourceRowDTO) []SourceRowDTO {
	if len(rows) == 0 {
		return nil
	}
	out := make([]SourceRowDTO, len(rows))
	for i := range rows {
		r := rows[i]
		out[i] = SourceRowDTO{
			CampaignID: r.CampaignID, Sub1: r.Sub1, Sub2: r.Sub2, Country: r.Country,
			Impressions: r.Impressions, Clicks: r.Clicks, Conversions: r.Conversions,
			SpendMicro: r.SpendMicro, RevenueMicro: r.RevenueMicro, ProfitMicro: r.ProfitMicro,
			CPAMicro: r.CPAMicro, ROIPct: r.ROIPct, CTR: r.CTR, IVTRate: r.IVTRate,
			QualityScore: r.QualityScore,
		}
	}
	return out
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
				QualityScore: reports.CalcQualityFromDrift(c.PacingDriftPct),
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

func (st *RoleService) GetCFODashboard(ctx context.Context, customerID uuid.UUID) (CFODashboardDTO, error) {
	if customerID == uuid.Nil {
		return CFODashboardDTO{}, st.host.ErrValidation("customer_id is required")
	}
	now := time.Now().UTC()
	from := now.Add(-30 * 24 * time.Hour)

	stmt, err := st.host.BuildStatement(ctx, customerID, from, now)
	if err != nil {
		return CFODashboardDTO{}, err
	}
	spendMicro := statementSpendMicro(stmt.Lines)
	feeMicro := statementFeeMicro(stmt.Lines)
	if stmt.TaxMicro > 0 {
		feeMicro += stmt.TaxMicro
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
		BilledMicro:          stmt.InvoiceTotalMicro,
		ARAgingMicro:         arAging,
		FeeTotalMicro:        feeMicro,
		DisputeExposureMicro: st.host.SumDisputeExposure(ctx, customerID),
		KPIs: MetricsBlockDTO{
			SpendMicro: spendMicro,
			Freshness: DataFreshnessDTO{
				AsOf:        now.Format(time.RFC3339),
				Consistency: "strong",
				Stale:       st.host.ClickHouseQuery() == nil,
			},
		},
	}, nil
}

func (st *RoleService) GetAccountantDashboard(ctx context.Context, customerID uuid.UUID) (AccountantDashboardDTO, error) {
	if customerID == uuid.Nil {
		return AccountantDashboardDTO{}, st.host.ErrValidation("customer_id is required")
	}
	now := time.Now().UTC()
	month := now.Format("2006-01")

	inv, err := st.host.GetInvariant(ctx, &customerID)
	if err != nil {
		return AccountantDashboardDTO{}, err
	}

	var countryCode, taxScheme string
	err = st.host.Pool().QueryRow(ctx, `
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

func (st *RoleService) GetFraudDashboard(ctx context.Context, customerID uuid.UUID) (FraudDashboardDTO, error) {
	now := time.Now().UTC()
	return st.GetFraudDashboardRange(ctx, customerID, now.Add(-7*24*time.Hour), now)
}

func (st *RoleService) GetFraudDashboardRange(ctx context.Context, customerID uuid.UUID, from, to time.Time) (FraudDashboardDTO, error) {
	if customerID == uuid.Nil {
		return FraudDashboardDTO{}, st.host.ErrValidation("customer_id is required")
	}
	if err := reports.ValidateChartRange(from, to); err != nil {
		return FraudDashboardDTO{}, err
	}
	now := time.Now().UTC()
	var silentRejectCount int
	err := st.host.Pool().QueryRow(ctx, `
		SELECT count(*)::int FROM campaigns
		WHERE customer_id = $1 AND silent_reject_enabled = TRUE AND deleted_at IS NULL`,
		domain.ToUUID(customerID),
	).Scan(&silentRejectCount)
	if err != nil {
		return FraudDashboardDTO{}, err
	}

	var labelsPending int
	if err := st.host.Pool().QueryRow(ctx, `
		SELECT count(*)::int FROM ml_manual_labels
		WHERE customer_id = $1 AND created_at >= $2`, domain.ToUUID(customerID), from).Scan(&labelsPending); err != nil {
		return FraudDashboardDTO{}, fmt.Errorf("count pending ml labels: %w", err)
	}

	mlSnap, err := st.host.FraudMLSnapshot(ctx)
	if err != nil {
		return FraudDashboardDTO{}, err
	}

	recentLabels, err := st.host.ListMLManualLabels(ctx, customerID, 5)
	if err != nil {
		return FraudDashboardDTO{}, fmt.Errorf("list recent ml labels: %w", err)
	}

	geoHints := st.fraudGeoHints(ctx, customerID, from, now)

	edge, err := st.host.FetchEdgeMetrics(ctx)
	if err != nil {
		return FraudDashboardDTO{}, fmt.Errorf("fetch edge metrics: %w", err)
	}
	campaignIDs, _ := st.reportHost.ListCustomerCampaignIDs(ctx, customerID)
	var series []DashboardSeriesPointDTO
	if st.host.ClickHouseQuery() != nil && st.reportHost != nil {
		clickhouseCtx, cancel := context.WithTimeout(ctx, st.host.ReportCHTimeout())
		series, _ = st.reportHost.QueryCustomerDashboardSeries(clickhouseCtx, customerID, campaignIDs, from, to)
		cancel()
	}
	return FraudDashboardDTO{
		CustomerID: customerID.String(),
		Period: PeriodDTO{
			From: from.Format(time.RFC3339),
			To:   to.Format(time.RFC3339),
		},
		Series:                series,
		SilentRejectCampaigns: silentRejectCount,
		LabelsPending:         labelsPending,
		EdgeBlockedFraud:      edge.Blocked["fraud_tier"],
		MLActiveVersionID:     mlSnap.VersionID,
		MLArtifactHash:        mlSnap.ArtifactHash,
		MLPrecision:           mlSnap.Precision,
		MLRecall:              mlSnap.Recall,
		MLDriftDetected:       mlSnap.DriftDetected,
		MLDriftSummary:        mlSnap.DriftSummary,
		MLEvalGeneratedAt:     mlSnap.EvalGeneratedAt,
		MLEvalStatus:          mlSnap.EvalStatus,
		MLEvalStale:           mlSnap.EvalStale,
		MLLabelMethod:         mlSnap.LabelMethod,
		MLShardsConsistent:    mlSnap.ShardsConsistent,
		FraudTierThresholds: FraudTierThresholdsDTO{
			Scope:      fraudTierThresholdScopePlatformDefault,
			PassMax:    int(domain.DefaultFraudThresholdPass),
			SuspectMax: int(domain.DefaultFraudThresholdSuspect),
			IVTMax:     int(domain.DefaultFraudThresholdIVT),
			BlockAbove: int(domain.DefaultFraudThresholdBlock),
		},
		GeoHints:     fraudGeoHintsFromReports(geoHints),
		RecentLabels: recentLabels,
	}, nil
}

func (st *RoleService) fraudGeoHints(ctx context.Context, customerID uuid.UUID, from, to time.Time) []reports.FraudGeoHintDTO {
	if st.host.ClickHouseQuery() == nil || st.reportHost == nil {
		return nil
	}
	campaignIDs, err := st.reportHost.ListCustomerCampaignIDs(ctx, customerID)
	if err != nil || len(campaignIDs) == 0 {
		return nil
	}
	clickhouseCtx, cancel := context.WithTimeout(ctx, st.host.ReportCHTimeout())
	defer cancel()
	hints, _ := st.reportHost.QueryWorstIVTCountries(clickhouseCtx, campaignIDs, from, to, 5)
	return hints
}

func fraudGeoHintsFromReports(hints []reports.FraudGeoHintDTO) []FraudGeoHintDTO {
	if len(hints) == 0 {
		return nil
	}
	out := make([]FraudGeoHintDTO, len(hints))
	for i := range hints {
		h := hints[i]
		out[i] = FraudGeoHintDTO{
			Country: h.Country, IVTRate: h.IVTRate, IVTEvents: h.IVTEvents,
			Clicks: h.Clicks, CampaignID: h.CampaignID,
		}
	}
	return out
}
