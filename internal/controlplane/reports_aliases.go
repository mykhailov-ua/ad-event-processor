package controlplane

import (
	"context"
	"time"

	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/reportjob"
	"ad-event-processor/internal/reports"

	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type (
	ReportJobRunner          = reportjob.ReportJobRunner
	ReportJobSpec            = reportjob.ReportJobSpec
	ReportJobStatusDTO       = reportjob.ReportJobStatusDTO
	ReportJobHTTPHandlers    = reportjob.HTTPHandlers
	ReportExportDeps         = reports.ReportExportDeps
	ExportDeps               = reportjob.ExportDeps
	ReportsHTTPHandlers      = reports.ReportsHTTPHandlers
	FraudEvidencePackDTO     = reports.FraudEvidencePackDTO
	DataFreshnessDTO         = reports.DataFreshnessDTO
	DataSourceFreshnessDTO   = reports.DataSourceFreshnessDTO
	DashboardSeriesPointDTO  = reports.DashboardSeriesPointDTO
	CustomerFraudOverviewDTO = reports.CustomerFraudOverviewDTO
	PlacementReportRowDTO    = reports.PlacementReportRowDTO
)

var (
	NewReportJobRunner         = reportjob.NewReportJobRunner
	NewReportScheduleWorker    = reportjob.NewReportScheduleWorker
	DefaultReportExportDirPath = reportjob.DefaultReportExportDirPath
)

const permCampaignsReadMasked = "campaigns:read:masked"

var reportClickHouseQueryTimeout = reports.ReportClickHouseQueryTimeout()

func ReportClickHouseQueryTimeout() time.Duration {
	return reports.ReportClickHouseQueryTimeout()
}

var (
	reportPermsFraudCustomer                = reports.ReportPermsFraudCustomer
	parseReportRange                        = reports.ParseReportRange
	queryPlacementReportRows                = reports.QueryPlacementReportRows
	queryPlacementIVTRates                  = reports.QueryPlacementIVTRates
	toPlacementReportRowDTO                 = reports.ToPlacementReportRowDTO
	reportMetricsKey                        = reports.ReportMetricsKey
	queryClickHouseCampaignDailyEventTotals = reports.QueryClickHouseCampaignDailyEventTotals
	parseReportRangeFromStrings             = reportjob.ParseReportRangeFromStrings
	campaignDailyTotalKey                   = reports.CampaignDailyTotalKey
	queryCustomerFraudOverview              = reports.QueryCustomerFraudOverview
	queryCustomerFraudDailySeries           = reports.QueryCustomerFraudDailySeries
	campaignDashboardFreshness              = reports.CampaignDashboardFreshness
)

var (
	QueryCampaignEconomicsCH = reports.QueryCampaignEconomicsCH
	ListCustomerCampaignIDs  = reports.ListCustomerCampaignIDs
	listCustomerCampaignIDs  = reports.ListCustomerCampaignIDs
	QueryWorstIVTSources     = reports.QueryWorstIVTSources
	QueryWorstIVTCountries   = reports.QueryWorstIVTCountries
	CalcROIPct               = reports.CalcROIPct
	CalcQualityFromDrift     = reports.CalcQualityFromDrift
	filterReportCatalog      = reports.FilterReportCatalog
	reportCatalogEntries     = reports.ReportCatalogEntries
	observeReportHandler     = reports.ObserveReportHandler
	liveReportExportKeys     = reports.LiveReportExportKeys
)

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
			CPAMicro: r.CPAMicro, ROIPct: r.ROIPct, CTR: r.CTR, IVTRate: r.IVTRate, QualityScore: r.QualityScore,
		}
	}
	return out
}

type (
	ForecastUnavailableResponse = reports.ForecastUnavailableResponse
	ForecastErrorDetail         = reports.ForecastErrorDetail
	FraudEvidenceTimelineRowDTO = reports.FraudEvidenceTimelineRowDTO
	FraudEvidenceFraudRowDTO    = reports.FraudEvidenceFraudRowDTO
)

var (
	writeBuyerFraudExportPreamble     = reports.WriteBuyerFraudExportPreamble
	buildSignedFraudEvidencePack      = reports.BuildSignedFraudEvidencePack
	verifyFraudEvidencePackSignature  = reports.VerifyFraudEvidencePackSignature
	campaignImportValidationReportKey = reportjob.CampaignImportValidationReportKey
)

func maskLevelFromContext(ctx context.Context) authz.MaskLevel {
	if snap, ok := authz.SnapshotFromContext(ctx); ok {
		return snap.Mask
	}
	return authz.MaskFull
}

func requestHasShardsRead(r *http.Request) bool {
	if snap, ok := authz.SnapshotFromContext(r.Context()); ok {
		return snap.Has("shards:read")
	}
	user, ok := GetUser(r.Context())
	if !ok {
		return false
	}
	return HasPermission(user.Role, PermShardsRead)
}

func portfolioFreshness(now time.Time, chQueryAvailable bool, chLag time.Duration) DataFreshnessDTO {
	return reports.PortfolioFreshness(now, chQueryAvailable, chLag)
}

func dataFreshnessFromClickHouse(ctx context.Context, clickhouseQuery *database.ClickHouseQuery) DataFreshnessDTO {
	return reports.DataFreshnessFromClickHouse(ctx, clickhouseQuery)
}

func validateChartRange(from, to time.Time) error {
	return reports.ValidateChartRange(from, to)
}

func buildCustomerFraudOverview(totalEvents, blockedEvents, silentRejectEvents int64, freshness DataFreshnessDTO) CustomerFraudOverviewDTO {
	return reports.BuildCustomerFraudOverview(totalEvents, blockedEvents, silentRejectEvents, freshness)
}

func attachInvalidSpendKPI(out *CustomerFraudOverviewDTO, blockedEvents, silentRejectEvents, totalEvents int64, spendMicros int64, attributionCoverage float64) {
	reports.AttachInvalidSpendKPI(out, blockedEvents, silentRejectEvents, totalEvents, spendMicros, attributionCoverage)
}

func queryCustomerDashboardSeries(ctx context.Context, pool *pgxpool.Pool, clickhouseQuery *database.ClickHouseQuery, customerID uuid.UUID, campaignIDs []uuid.UUID, from, to time.Time) ([]DashboardSeriesPointDTO, error) {
	return reports.QueryCustomerDashboardSeries(ctx, pool, clickhouseQuery, customerID, campaignIDs, from, to)
}

func scrubCustomerFraudEvidencePack(pack FraudEvidencePackDTO) FraudEvidencePackDTO {
	return reports.ScrubCustomerFraudEvidencePack(pack)
}

func computeAttributionCoverage(totalEvents, attributedEvents int64) float64 {
	return reports.ComputeAttributionCoverage(totalEvents, attributedEvents)
}

type campaignStatsAdapter struct {
	svc *Service
}

func (a campaignStatsAdapter) GetCampaignStats(ctx context.Context, campaignID uuid.UUID, from, to time.Time, granularity string) (reports.CampaignStatsDTO, error) {
	report, err := a.svc.GetCampaignStats(ctx, campaignID, from, to, granularity)
	if err != nil {
		return reports.CampaignStatsDTO{}, err
	}
	out := reports.CampaignStatsDTO{
		CampaignID:   report.CampaignID,
		CurrentSpend: report.CurrentSpend,
		Granularity:  report.Granularity,
		From:         report.From,
		To:           report.To,
		Stale:        report.Stale,
		Source:       report.Source,
		Consistency:  report.Consistency,
		Metrics: reports.CampaignMetricsDTO{
			Impressions: report.Metrics.Impressions,
			Clicks:      report.Metrics.Clicks,
			Conversions: report.Metrics.Conversions,
		},
	}
	for _, h := range report.Hourly {
		out.Hourly = append(out.Hourly, reports.CampaignHourlyBucketDTO{
			Hour: h.Hour, Impressions: h.Impressions, Clicks: h.Clicks, Conversions: h.Conversions,
		})
	}
	for _, d := range report.Daily {
		out.Daily = append(out.Daily, reports.CampaignDailyBucketDTO{
			Day: d.Day, Impressions: d.Impressions, Clicks: d.Clicks, Conversions: d.Conversions,
		})
	}
	return out, nil
}

type campaignForecasterAdapter struct {
	svc *Service
}

func (a campaignForecasterAdapter) ForecastCampaign(ctx context.Context, in reports.CampaignForecastInput) (reports.CampaignForecastDTO, error) {
	out, err := a.svc.ForecastCampaign(ctx, CampaignForecastInput{
		CustomerID:       in.CustomerID,
		BudgetLimitMicro: in.BudgetLimitMicro,
		TargetCountries:  in.TargetCountries,
		DaypartHours:     in.DaypartHours,
		StartAt:          in.StartAt,
		EndAt:            in.EndAt,
		PacingMode:       in.PacingMode,
		Timezone:         in.Timezone,
	})
	if err != nil {
		return reports.CampaignForecastDTO{}, err
	}
	dto := reports.CampaignForecastDTO{
		ImpressionsP50: out.ImpressionsP50,
		ImpressionsP90: out.ImpressionsP90,
		LowConfidence:  out.LowConfidence,
	}
	if out.Advisory != nil {
		dto.Advisory = &reports.ForecastAdvisory{
			Code: out.Advisory.Code, Message: out.Advisory.Message, SuggestedPacing: out.Advisory.SuggestedPacing,
		}
	}
	for _, p := range out.SpendCurve {
		dto.SpendCurve = append(dto.SpendCurve, reports.SpendCurvePoint{
			Hour: p.Hour, SpendMicro: p.SpendMicro, Impressions: p.Impressions,
		})
	}
	return dto, nil
}

type buyerPortfolioAdapter struct {
	svc *Service
}

func (a buyerPortfolioAdapter) GetBuyerPortfolio(ctx context.Context, customerID uuid.UUID) (reports.BuyerPortfolioDTO, error) {
	p, err := a.svc.GetBuyerPortfolio(ctx, customerID)
	return buyerPortfolioToReports(p), err
}

func (a buyerPortfolioAdapter) GetBuyerPortfolioRange(ctx context.Context, customerID uuid.UUID, from, to time.Time) (reports.BuyerPortfolioDTO, error) {
	p, err := a.svc.GetBuyerPortfolioRange(ctx, customerID, from, to)
	return buyerPortfolioToReports(p), err
}

func buyerPortfolioToReports(p BuyerPortfolioDTO) reports.BuyerPortfolioDTO {
	out := reports.BuyerPortfolioDTO{
		CustomerID:     p.CustomerID,
		Active:         p.Active,
		Paused:         p.Paused,
		Archived:       p.Archived,
		Impressions7d:  p.Impressions7d,
		Clicks7d:       p.Clicks7d,
		OverspendCount: p.OverspendCount,
	}
	for _, a := range p.Attention {
		out.Attention = append(out.Attention, reports.BuyerAttentionDTO{ID: a.ID, Name: a.Name, Reason: a.Reason})
	}
	for _, c := range p.Campaigns {
		out.Campaigns = append(out.Campaigns, reports.BuyerCampaignPortfolioRowDTO{
			ID: c.ID, Name: c.Name, Status: c.Status, PacingMode: c.PacingMode,
			Impressions7d: c.Impressions7d, Clicks7d: c.Clicks7d,
			SpendMicro: c.SpendMicro, BudgetMicro: c.BudgetMicro,
			UtilizationPct: c.UtilizationPct, PacingDriftPct: c.PacingDriftPct,
			EstimatedPacingDriftPct: c.EstimatedPacingDriftPct,
			OverspendRisk:           c.OverspendRisk, MarginBreach: c.MarginBreach,
		})
	}
	if p.Fraud != nil {
		out.Fraud = p.Fraud
	}
	return out
}
