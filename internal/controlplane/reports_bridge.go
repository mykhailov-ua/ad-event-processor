// reports_bridge: delegates admin report handlers to reports/* with CH query gate from Service.
package controlplane

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	ctrlhttp "ad-event-processor/internal/control/http"

	"ad-event-processor/internal/campaign/importexport"
	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/internal/dashboardadmin"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/reportjob"
	"ad-event-processor/internal/reports"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func ReportClickHouseQueryTimeout() time.Duration {
	return reports.ReportClickHouseQueryTimeout()
}

func requestHasShardsRead(r *http.Request) bool {
	if snap, ok := authz.SnapshotFromContext(r.Context()); ok {
		return snap.Has("shards:read")
	}
	user, ok := GetUser(r.Context())
	if !ok {
		return false
	}
	return ctrlhttp.HasPermission(user.Role, ctrlhttp.PermShardsRead)
}

func portfolioFreshness(now time.Time, chQueryAvailable bool, chLag time.Duration) reports.DataFreshnessDTO {
	return reports.PortfolioFreshness(now, chQueryAvailable, chLag)
}

func queryCustomerDashboardSeries(ctx context.Context, pool *pgxpool.Pool, clickhouseQuery *database.ClickHouseQuery, customerID uuid.UUID, campaignIDs []uuid.UUID, from, to time.Time, granularity reports.ChartGranularity) ([]reports.DashboardSeriesPointDTO, error) {
	return reports.QueryCustomerDashboardSeries(ctx, pool, clickhouseQuery, customerID, campaignIDs, from, to, granularity)
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
	return reports.ForecastCampaign(ctx, forecastHost{svc: a.svc}, in)
}

type forecastHost struct {
	svc *Service
}

func (h forecastHost) ForecastPool() *pgxpool.Pool {
	return h.svc.GetPool()
}

func (h forecastHost) ForecastClickHouseQuery() *database.ClickHouseQuery {
	return h.svc.clickhouseQuery
}

type buyerPortfolioAdapter struct {
	svc *Service
}

func (a buyerPortfolioAdapter) GetBuyerPortfolio(ctx context.Context, customerID uuid.UUID) (reports.BuyerPortfolioDTO, error) {
	p, err := a.svc.GetBuyerPortfolio(ctx, customerID)
	return buyerPortfolioToReports(p), err
}

func (a buyerPortfolioAdapter) GetBuyerPortfolioRange(ctx context.Context, customerID uuid.UUID, campaignFilter *uuid.UUID, from, to time.Time, seriesGranularity reports.ChartGranularity) (reports.BuyerPortfolioDTO, error) {
	p, err := a.svc.GetBuyerPortfolioRange(ctx, customerID, campaignFilter, from, to, seriesGranularity)
	return buyerPortfolioToReports(p), err
}

func buyerPortfolioToReports(p dashboardadmin.BuyerPortfolioDTO) reports.BuyerPortfolioDTO {
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

func (s *Service) InitReportJobRunner(exportDir string) *reportjob.ReportJobRunner {
	if s == nil {
		return nil
	}
	if exportDir == "" {
		exportDir = reportjob.DefaultReportExportDirPath()
	}
	s.workerMutex.Lock()
	defer s.workerMutex.Unlock()
	if s.reportJobRunner == nil {
		var packSecret []byte
		if s.cfg != nil {
			packSecret = []byte(s.cfg.FraudEvidencePackHMACSecret)
		}
		exportDeps := reports.ReportExportDeps{
			Pool:                        s.pool,
			ClickHouseQuery:             s.clickhouseQuery,
			BuyerPortfolio:              buyerPortfolioAdapter{svc: s},
			FraudEvidencePackHMACSecret: packSecret,
		}
		s.reportJobRunner = reportjob.NewReportJobRunner(exportDir, reportjob.ExportDeps{
			Pool: s.pool,
			WriteReport: func(ctx context.Context, path string, spec reportjob.ReportJobSpec) error {
				return reports.WriteReportExport(ctx, exportDeps, path, spec)
			},
			WriteCampaignImportValidation: importexport.WriteCampaignImportValidationJSON,
		})
	}
	return s.reportJobRunner
}

func (s *Service) ReportJobRunner() *reportjob.ReportJobRunner {
	if s == nil {
		return nil
	}
	s.workerMutex.Lock()
	defer s.workerMutex.Unlock()
	return s.reportJobRunner
}

func (s *Service) StartReportJobWorker(ctx context.Context) {
	runner := s.ReportJobRunner()
	if runner == nil || !runner.PgEnabled() {
		return
	}
	s.StartBackgroundWorker(func() {
		runner.StartWorker(ctx)
	})
	slog.Info("report job worker starting")
}
