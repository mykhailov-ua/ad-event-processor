package controlplane

import (
	"context"
	"time"

	"ad-event-processor/internal/billingadmin"
	"ad-event-processor/internal/dashboardadmin"
	"ad-event-processor/internal/fraudadmin"
	"ad-event-processor/internal/opsadmin"
	"ad-event-processor/internal/reports"

	"github.com/google/uuid"
)

var (
	_ dashboardadmin.PortfolioHost           = (*Service)(nil)
	_ dashboardadmin.RoleHost                = (*Service)(nil)
	_ dashboardadmin.RoleReportHost          = (*Service)(nil)
	_ dashboardadmin.CampaignHost            = (*Service)(nil)
	_ dashboardadmin.PublisherHost           = (*Service)(nil)
	_ dashboardadmin.BuyerPortfolioReader    = (*Service)(nil)
	_ dashboardadmin.CampaignDashboardReader = (*Service)(nil)
	_ dashboardadmin.RoleDashboardReader     = (*Service)(nil)
	_ dashboardadmin.PublisherReader         = (*Service)(nil)
)

func (s *Service) ReportCHTimeout() time.Duration { return ReportClickHouseQueryTimeout() }

func (s *Service) ClickHouseIngestionLag(ctx context.Context) (time.Duration, error) {
	return s.clickHouseIngestionLag(ctx)
}

func (s *Service) ClickHouseLag(ctx context.Context) time.Duration {
	lag, _ := s.clickHouseIngestionLag(ctx)
	return lag
}

func (s *Service) BatchCampaignMarginBreach(ctx context.Context, campaignIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
	return s.batchCampaignMarginBreach(ctx, campaignIDs)
}

func (s *Service) compositeBillingReads() *billingadmin.CompositeReadService {
	return billingadmin.NewCompositeReadService(s.GetPool(), s.cfg)
}

func (s *Service) BuildStatement(ctx context.Context, customerID uuid.UUID, from, to time.Time) (dashboardadmin.BillingStatement, error) {
	stmt, err := s.compositeBillingReads().BuildStatement(ctx, customerID, from, to)
	if err != nil {
		return dashboardadmin.BillingStatement{}, err
	}
	return dashboardadmin.BillingStatementFromLines(stmt.Lines, stmt.TaxBreakdown.TaxMicro, stmt.ClosingBalanceMicro, stmt.Reconciliation.InvoiceTotalMicro), nil
}

func (s *Service) GetInvariant(ctx context.Context, customerID *uuid.UUID) (dashboardadmin.BillingInvariant, error) {
	inv, err := s.compositeBillingReads().GetInvariant(ctx, customerID)
	if err != nil {
		return dashboardadmin.BillingInvariant{}, err
	}
	return dashboardadmin.BillingInvariantFrom(inv.OK, inv.DiffMicro), nil
}

func (s *Service) SumDisputeExposure(ctx context.Context, customerID uuid.UUID) int64 {
	return dashboardadmin.SumDisputeExposure(ctx, s.GetPool(), customerID)
}

func (s *Service) FraudMLSnapshot(ctx context.Context) (dashboardadmin.FraudMLSnapshot, error) {
	snap, err := fraudadmin.BuildFraudMLSnapshot(ctx, s)
	if err != nil {
		return dashboardadmin.FraudMLSnapshot{}, err
	}
	return dashboardadmin.FraudMLSnapshotFrom(snap), nil
}

func (s *Service) ListMLManualLabels(ctx context.Context, customerID uuid.UUID, limit int) ([]dashboardadmin.MLManualLabelDTO, error) {
	rows, _, err := s.ListMLManualLabelsForCustomer(ctx, customerID, limit, 0)
	if err != nil {
		return nil, err
	}
	return dashboardadmin.MLManualLabelsFrom(rows), nil
}

func (s *Service) FetchEdgeMetrics(ctx context.Context) (dashboardadmin.EdgeMetricsPanelDTO, error) {
	panel, err := opsadmin.FetchEdgeMetrics(ctx)
	if err != nil {
		return dashboardadmin.EdgeMetricsPanelDTO{}, err
	}
	return dashboardadmin.EdgeMetricsFrom(panel), nil
}

func (s *Service) ListCustomerCampaignIDs(ctx context.Context, customerID uuid.UUID) ([]uuid.UUID, error) {
	return reports.ListCustomerCampaignIDs(ctx, s.GetPool(), customerID)
}

func (s *Service) QueryWorstIVTSources(ctx context.Context, campaignIDs []uuid.UUID, from, to time.Time, limit int) ([]reports.SourceRowDTO, error) {
	return reports.QueryWorstIVTSources(ctx, s.clickhouseQuery, campaignIDs, from, to, limit)
}

func (s *Service) QueryWorstIVTCountries(ctx context.Context, campaignIDs []uuid.UUID, from, to time.Time, limit int) ([]reports.FraudGeoHintDTO, error) {
	return reports.QueryWorstIVTCountries(ctx, s.clickhouseQuery, campaignIDs, from, to, limit)
}

func (s *Service) QueryCustomerDashboardSeries(ctx context.Context, customerID uuid.UUID, campaignIDs []uuid.UUID, from, to time.Time) ([]reports.DashboardSeriesPointDTO, error) {
	return queryCustomerDashboardSeries(ctx, s.GetPool(), s.clickhouseQuery, customerID, campaignIDs, from, to)
}

func (s *Service) buyerPortfolio() *dashboardadmin.Portfolio {
	if s == nil {
		return nil
	}
	return dashboardadmin.NewPortfolio(s)
}

func (s *Service) roleDashboard() *dashboardadmin.RoleService {
	if s == nil {
		return nil
	}
	return dashboardadmin.NewRoleService(s, s)
}

func (s *Service) campaignDashboard() *dashboardadmin.CampaignService {
	if s == nil {
		return nil
	}
	return dashboardadmin.NewCampaignService(s)
}

func (s *Service) dashboardPublisherService() *dashboardadmin.PublisherService {
	if s == nil {
		return nil
	}
	return dashboardadmin.NewPublisherService(s)
}

func (s *Service) GetBuyerPortfolio(ctx context.Context, customerID uuid.UUID) (dashboardadmin.BuyerPortfolioDTO, error) {
	return s.buyerPortfolio().GetBuyerPortfolio(ctx, customerID)
}

func (s *Service) GetBuyerPortfolioRange(ctx context.Context, customerID uuid.UUID, campaignFilter *uuid.UUID, from, to time.Time) (dashboardadmin.BuyerPortfolioDTO, error) {
	return s.buyerPortfolio().GetBuyerPortfolioRange(ctx, customerID, campaignFilter, from, to)
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
	return s.dashboardPublisherService().ValidateSupplyFiles(ctx)
}
