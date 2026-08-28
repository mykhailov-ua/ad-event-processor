package controlplane

import (
	"ad-event-processor/internal/billingadmin"
	"ad-event-processor/internal/campaign"
	"ad-event-processor/internal/dashboardadmin"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/fraudadmin"
	"ad-event-processor/internal/opsadmin"
	"ad-event-processor/internal/reports"
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type portfolioHost struct{ svc *Service }

func (h portfolioHost) Pool() *pgxpool.Pool { return h.svc.GetPool() }

func (h portfolioHost) ClickHouseQuery() *database.ClickHouseQuery { return h.svc.clickhouseQuery }

func (h portfolioHost) ClickHouseIngestionLag(ctx context.Context) (time.Duration, error) {
	return h.svc.clickHouseIngestionLag(ctx)
}

func (h portfolioHost) ReportCHTimeout() time.Duration { return ReportClickHouseQueryTimeout() }

func (h portfolioHost) ErrValidation(msg string) error { return errValidation(msg) }

func (h portfolioHost) ListCampaigns(ctx context.Context, customerID uuid.UUID, status string, limit, offset int32) ([]campaign.CampaignDTO, int64, error) {
	return h.svc.ListCampaigns(ctx, customerID, status, limit, offset)
}

func (h portfolioHost) BatchCampaignMarginBreach(ctx context.Context, campaignIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
	return h.svc.batchCampaignMarginBreach(ctx, campaignIDs)
}

type dashboardRoleHost struct{ svc *Service }

func (h dashboardRoleHost) ErrValidation(msg string) error { return errValidation(msg) }

func (h dashboardRoleHost) Pool() *pgxpool.Pool { return h.svc.GetPool() }

func (h dashboardRoleHost) ClickHouseQuery() *database.ClickHouseQuery { return h.svc.clickhouseQuery }

func (h dashboardRoleHost) ReportCHTimeout() time.Duration { return ReportClickHouseQueryTimeout() }

func (h dashboardRoleHost) GetBuyerPortfolio(ctx context.Context, customerID uuid.UUID) (dashboardadmin.BuyerPortfolioDTO, error) {
	return h.svc.buyerPortfolio().GetBuyerPortfolio(ctx, customerID)
}

func (h dashboardRoleHost) compositeReads() *billingadmin.CompositeReadService {
	return billingadmin.NewCompositeReadService(h.svc.GetPool(), h.svc.cfg)
}

func (h dashboardRoleHost) BuildStatement(ctx context.Context, customerID uuid.UUID, from, to time.Time) (dashboardadmin.BillingStatement, error) {
	stmt, err := h.compositeReads().BuildStatement(ctx, customerID, from, to)
	if err != nil {
		return dashboardadmin.BillingStatement{}, err
	}
	return dashboardadmin.BillingStatement{
		Lines:               stmt.Lines,
		TaxMicro:            stmt.TaxBreakdown.TaxMicro,
		ClosingBalanceMicro: stmt.ClosingBalanceMicro,
		InvoiceTotalMicro:   stmt.Reconciliation.InvoiceTotalMicro,
	}, nil
}

func (h dashboardRoleHost) GetInvariant(ctx context.Context, customerID *uuid.UUID) (dashboardadmin.BillingInvariant, error) {
	inv, err := h.compositeReads().GetInvariant(ctx, customerID)
	if err != nil {
		return dashboardadmin.BillingInvariant{}, err
	}
	return dashboardadmin.BillingInvariant{OK: inv.OK, DiffMicro: inv.DiffMicro}, nil
}

func (h dashboardRoleHost) SumDisputeExposure(ctx context.Context, customerID uuid.UUID) int64 {
	if h.svc == nil || h.svc.GetPool() == nil || customerID == uuid.Nil {
		return 0
	}
	var exposure int64
	_ = h.svc.GetPool().QueryRow(ctx, `
		SELECT COALESCE(SUM(d.amount_micro), 0)
		FROM payment.payment_disputes d
		JOIN payment.payment_intents i ON i.id = d.payment_intent_id
		WHERE i.customer_id = $1
		  AND d.status IN ('OPEN', 'FUNDS_WITHDRAWN')`,
		domain.ToUUID(customerID),
	).Scan(&exposure)
	return exposure
}

func (h dashboardRoleHost) FraudMLSnapshot(ctx context.Context) (dashboardadmin.FraudMLSnapshot, error) {
	snap, err := fraudadmin.BuildFraudMLSnapshot(ctx, fraudMLSnapshotHost{svc: h.svc})
	if err != nil {
		return dashboardadmin.FraudMLSnapshot{}, err
	}
	return dashboardadmin.FraudMLSnapshot{
		VersionID: snap.VersionID, ArtifactHash: snap.ArtifactHash, Precision: snap.Precision, Recall: snap.Recall,
		DriftDetected: snap.DriftDetected, DriftSummary: snap.DriftSummary, EvalGeneratedAt: snap.EvalGeneratedAt,
		EvalStatus: snap.EvalStatus, EvalStale: snap.EvalStale, LabelMethod: snap.LabelMethod, ShardsConsistent: snap.ShardsConsistent,
	}, nil
}

func (h dashboardRoleHost) ListMLManualLabels(ctx context.Context, customerID uuid.UUID, limit int) ([]dashboardadmin.MLManualLabelDTO, error) {
	rows, err := h.svc.ListMLManualLabelsForCustomer(ctx, customerID, limit)
	if err != nil {
		return nil, err
	}
	out := make([]dashboardadmin.MLManualLabelDTO, len(rows))
	for i := range rows {
		out[i] = dashboardadmin.MLManualLabelDTO{
			IPHash: rows[i].IPHash, Label: rows[i].Label, Reason: rows[i].Reason,
			Source: rows[i].Source, CreatedAt: rows[i].CreatedAt,
		}
	}
	return out, nil
}

func (h dashboardRoleHost) FetchEdgeMetrics(ctx context.Context) (dashboardadmin.EdgeMetricsPanelDTO, error) {
	panel, err := opsadmin.FetchEdgeMetrics(ctx)
	if err != nil {
		return dashboardadmin.EdgeMetricsPanelDTO{}, err
	}
	return dashboardadmin.EdgeMetricsPanelDTO{
		UpdatedAt: panel.UpdatedAt, IngressH1: panel.IngressH1, IngressH2: panel.IngressH2, IngressH3: panel.IngressH3,
		BodyStream: panel.BodyStream, BodyPeek: panel.BodyPeek, BodyRead: panel.BodyRead,
		Blocked: panel.Blocked, TarpitTotal: panel.TarpitTotal, BlacklistStale: panel.BlacklistStale,
	}, nil
}

type dashboardReportHost struct{ svc *Service }

func (h dashboardReportHost) ListCustomerCampaignIDs(ctx context.Context, customerID uuid.UUID) ([]uuid.UUID, error) {
	return reports.ListCustomerCampaignIDs(ctx, h.svc.GetPool(), customerID)
}

func (h dashboardReportHost) QueryWorstIVTSources(ctx context.Context, campaignIDs []uuid.UUID, from, to time.Time, limit int) ([]reports.SourceRowDTO, error) {
	return reports.QueryWorstIVTSources(ctx, h.svc.clickhouseQuery, campaignIDs, from, to, limit)
}

func (h dashboardReportHost) QueryWorstIVTCountries(ctx context.Context, campaignIDs []uuid.UUID, from, to time.Time, limit int) ([]reports.FraudGeoHintDTO, error) {
	return reports.QueryWorstIVTCountries(ctx, h.svc.clickhouseQuery, campaignIDs, from, to, limit)
}

func (h dashboardReportHost) QueryCustomerDashboardSeries(ctx context.Context, customerID uuid.UUID, campaignIDs []uuid.UUID, from, to time.Time) ([]reports.DashboardSeriesPointDTO, error) {
	return queryCustomerDashboardSeries(ctx, h.svc.GetPool(), h.svc.clickhouseQuery, customerID, campaignIDs, from, to)
}

type dashboardCampaignHost struct{ svc *Service }

func (h dashboardCampaignHost) ErrValidation(msg string) error { return errValidation(msg) }

func (h dashboardCampaignHost) Pool() *pgxpool.Pool { return h.svc.GetPool() }

func (h dashboardCampaignHost) ClickHouseQuery() *database.ClickHouseQuery {
	return h.svc.clickhouseQuery
}

func (h dashboardCampaignHost) ReportCHTimeout() time.Duration { return ReportClickHouseQueryTimeout() }

func (h dashboardCampaignHost) ClickHouseLag(ctx context.Context) time.Duration {
	lag, _ := h.svc.clickHouseIngestionLag(ctx)
	return lag
}

func (h dashboardCampaignHost) MapCampaignNotFound(err error) error {
	return mapNotFound(err, ErrCampaignNotFound)
}

type dashboardPublisherHost struct{ svc *Service }

func (h dashboardPublisherHost) Pool() *pgxpool.Pool { return h.svc.GetPool() }

func (h dashboardPublisherHost) ClickHouseQuery() *database.ClickHouseQuery {
	return h.svc.clickhouseQuery
}

func (h dashboardPublisherHost) ReportCHTimeout() time.Duration {
	return ReportClickHouseQueryTimeout()
}

func (h dashboardPublisherHost) BuildSellersJSON(ctx context.Context) ([]byte, error) {
	return h.svc.BuildSellersJSON(ctx)
}

func (h dashboardPublisherHost) BuildAdsTxt(ctx context.Context) (string, error) {
	return h.svc.BuildAdsTxt(ctx)
}

func (s *Service) buyerPortfolio() *dashboardadmin.Portfolio {
	if s == nil {
		return nil
	}
	return dashboardadmin.NewPortfolio(portfolioHost{svc: s})
}

func (s *Service) roleDashboard() *dashboardadmin.RoleService {
	if s == nil {
		return nil
	}
	return dashboardadmin.NewRoleService(dashboardRoleHost{svc: s}, dashboardReportHost{svc: s})
}

func (s *Service) campaignDashboard() *dashboardadmin.CampaignService {
	if s == nil {
		return nil
	}
	return dashboardadmin.NewCampaignService(dashboardCampaignHost{svc: s})
}

func (s *Service) dashboardPublisherService() *dashboardadmin.PublisherService {
	if s == nil {
		return nil
	}
	return dashboardadmin.NewPublisherService(dashboardPublisherHost{svc: s})
}

func (s *Service) GetBuyerPortfolio(ctx context.Context, customerID uuid.UUID) (dashboardadmin.BuyerPortfolioDTO, error) {
	return s.buyerPortfolio().GetBuyerPortfolio(ctx, customerID)
}

func (s *Service) GetBuyerPortfolioRange(ctx context.Context, customerID uuid.UUID, from, to time.Time) (dashboardadmin.BuyerPortfolioDTO, error) {
	return s.buyerPortfolio().GetBuyerPortfolioRange(ctx, customerID, from, to)
}

func (s *Service) GetCampaignDashboard(ctx context.Context, campaignID uuid.UUID) (dashboardadmin.CampaignDashboardDTO, error) {
	return s.campaignDashboard().GetCampaignDashboard(ctx, campaignID)
}

func (s *Service) GetAdOpsDashboard(ctx context.Context, customerID uuid.UUID) (dashboardadmin.AdOpsDashboardDTO, error) {
	return s.roleDashboard().GetAdOpsDashboard(ctx, customerID)
}

func (s *Service) GetCFODashboard(ctx context.Context, customerID uuid.UUID) (dashboardadmin.CFODashboardDTO, error) {
	return s.roleDashboard().GetCFODashboard(ctx, customerID)
}

func (s *Service) GetAccountantDashboard(ctx context.Context, customerID uuid.UUID) (dashboardadmin.AccountantDashboardDTO, error) {
	return s.roleDashboard().GetAccountantDashboard(ctx, customerID)
}

func (s *Service) GetFraudDashboard(ctx context.Context, customerID uuid.UUID) (dashboardadmin.FraudDashboardDTO, error) {
	return s.roleDashboard().GetFraudDashboard(ctx, customerID)
}

func (s *Service) GetFraudDashboardRange(ctx context.Context, customerID uuid.UUID, from, to time.Time) (dashboardadmin.FraudDashboardDTO, error) {
	return s.roleDashboard().GetFraudDashboardRange(ctx, customerID, from, to)
}

func (s *Service) ResolvePublisherBind(ctx context.Context, userID uuid.UUID) (dashboardadmin.PublisherBind, error) {
	return s.dashboardPublisherService().ResolvePublisherBind(ctx, userID)
}

func (s *Service) GetPublisherDashboard(ctx context.Context, bind dashboardadmin.PublisherBind, from, to time.Time) (dashboardadmin.PublisherDashboardDTO, error) {
	return s.dashboardPublisherService().GetPublisherDashboard(ctx, bind, from, to)
}

func (s *Service) ListPublisherStatements(ctx context.Context, bind dashboardadmin.PublisherBind, from, to time.Time, limit, offset int32) ([]dashboardadmin.PublisherStatementDTO, int64, error) {
	return s.dashboardPublisherService().ListPublisherStatements(ctx, bind, from, to, limit, offset)
}

func (s *Service) ValidateSupplyFiles(ctx context.Context) (dashboardadmin.SupplyValidationReport, error) {
	report, err := s.dashboardPublisherService().ValidateSupplyFiles(ctx)
	if err != nil {
		return dashboardadmin.SupplyValidationReport{}, err
	}
	return report, nil
}

var (
	_ dashboardadmin.BuyerPortfolioReader    = (*Service)(nil)
	_ dashboardadmin.CampaignDashboardReader = (*Service)(nil)
	_ dashboardadmin.RoleDashboardReader     = (*Service)(nil)
	_ dashboardadmin.PublisherReader         = (*Service)(nil)
)
