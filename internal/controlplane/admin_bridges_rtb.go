package controlplane

import (
	"ad-event-processor/internal/campaign"
	"ad-event-processor/internal/config"
	"ad-event-processor/internal/dashboardadmin"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/dedup"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/fraudadmin"
	"ad-event-processor/internal/nodeadmin"
	"ad-event-processor/internal/opsadmin"
	"ad-event-processor/internal/outbox"
	"ad-event-processor/internal/reconciliation"
	"ad-event-processor/internal/reports"
	"ad-event-processor/internal/rtbadmin"
	"ad-event-processor/internal/shardadmin"
	"ad-event-processor/internal/supply"
	"ad-event-processor/pkg/coldpath"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type rtbAdminHost struct {
	svc *Service
}

type rtbRuntimeConfig struct {
	cfg *config.Config
}

func (s *Service) RtbAdminService() *rtbadmin.Service {
	if s == nil {
		return nil
	}
	return rtbadmin.NewService(rtbAdminHost{svc: s})
}

func (h rtbAdminHost) Pool() *pgxpool.Pool { return h.svc.GetPool() }

func (h rtbAdminHost) Config() *config.Config {
	if h.svc == nil {
		return nil
	}
	return h.svc.cfg
}

func (h rtbAdminHost) ErrValidation(msg string) error { return errValidation(msg) }

func (h rtbAdminHost) ActorUserID(ctx context.Context) uuid.UUID {
	if u, ok := GetUser(ctx); ok {
		return u.UserID
	}
	return uuid.Nil
}

func (h rtbAdminHost) AuditCreateRtbDeal(ctx context.Context, q db.Querier, adminID uuid.UUID, dealID string) {
	h.svc.AuditLog(ctx, q, adminID, "CREATE_RTB_DEAL", "rtb_deal", nil, map[string]string{"deal_id": dealID}, nil)
}

func (h rtbAdminHost) AuditUpdateRtbDeal(ctx context.Context, q db.Querier, adminID uuid.UUID, id int64, dealID string) {
	h.svc.AuditLog(ctx, q, adminID, "UPDATE_RTB_DEAL", "rtb_deal", nil, map[string]any{"id": id, "deal_id": dealID}, nil)
}

func (h rtbAdminHost) AuditDeleteRtbDeal(ctx context.Context, q db.Querier, adminID uuid.UUID, id int64, dealID string) {
	h.svc.AuditLog(ctx, q, adminID, "DELETE_RTB_DEAL", "rtb_deal", nil, map[string]any{"id": id, "deal_id": dealID}, nil)
}

func (h rtbAdminHost) EnqueueRtbCatalogReload(ctx context.Context, q db.Querier, trigger string) error {
	payload, err := coldpath.MarshalOutbox(outbox.RtbCatalogReloadPayload{Trigger: trigger})
	if err != nil {
		return err
	}
	_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
		EventType: "RELOAD_RTB_CATALOG",
		Payload:   payload,
	})
	return err
}

func (h rtbAdminHost) UpdateSettings(ctx context.Context, patch map[string]string) error {
	return h.svc.UpdateSettings(ctx, patch)
}

func (h rtbAdminHost) GetSettings(ctx context.Context) (map[string]string, error) {
	return h.svc.GetSettings(ctx)
}

func (h rtbAdminHost) SimulateBidShade(ctx context.Context, in domain.RtbBidShadeInput) (domain.RtbBidShadeOutput, error) {
	if h.svc == nil || h.svc.rtbBidShadeSim == nil {
		return domain.RtbBidShadeOutput{}, fmt.Errorf("rtb bid shade simulator not configured")
	}
	return h.svc.rtbBidShadeSim(ctx, h.svc.GetPool(), h.svc.cfg, in)
}

func (h rtbAdminHost) FloorsPool() *pgxpool.Pool { return h.svc.GetPool() }

func (h rtbAdminHost) FloorsConfig() *config.Config {
	if h.svc == nil {
		return nil
	}
	return h.svc.cfg
}

func (h rtbAdminHost) FloorsClickHouse() *database.ClickHouseQuery {
	if h.svc == nil {
		return nil
	}
	return h.svc.clickhouseQuery
}

func (h rtbAdminHost) FloorsRedisShards() []redis.UniversalClient {
	if h.svc == nil {
		return nil
	}
	return h.svc.redisShards
}

func (h rtbAdminHost) FloorsEnqueueRtbCatalogReload(ctx context.Context, q db.Querier, trigger string) error {
	return h.EnqueueRtbCatalogReload(ctx, q, trigger)
}

func (s *Service) ListRtbDeals(ctx context.Context) ([]RtbDealDTO, error) {
	return s.RtbAdminService().ListRtbDeals(ctx)
}

func (s *Service) GetRtbDeal(ctx context.Context, id int64) (RtbDealDTO, error) {
	return s.RtbAdminService().GetRtbDeal(ctx, id)
}

func (s *Service) CreateRtbDeal(ctx context.Context, spec RtbDealCreateSpec) (RtbDealDTO, error) {
	return s.RtbAdminService().CreateRtbDeal(ctx, spec)
}

func (s *Service) UpdateRtbDeal(ctx context.Context, id int64, spec RtbDealUpdateSpec) (RtbDealDTO, error) {
	return s.RtbAdminService().UpdateRtbDeal(ctx, id, spec)
}

func (s *Service) DeleteRtbDeal(ctx context.Context, id int64) error {
	return s.RtbAdminService().DeleteRtbDeal(ctx, id)
}

func (s *Service) SetRtbMode(ctx context.Context, mode string) error {
	return s.RtbAdminService().SetRtbMode(ctx, mode)
}

func (s *Service) GetRtbMode(ctx context.Context) string {
	return s.RtbAdminService().GetRtbMode(ctx)
}

func (s *Service) SimulateRtbBidShade(ctx context.Context, req RtbBidShadeRequest) (RtbBidShadeResponse, error) {
	out, err := s.RtbAdminService().SimulateRtbBidShade(ctx, rtbadmin.BidShadeRequest{
		GeoHash: req.GeoHash, DeviceType: req.DeviceType, CategoryMask: req.CategoryMask, MinBidMicro: req.MinBidMicro,
	})
	if err != nil {
		return RtbBidShadeResponse{}, err
	}
	return RtbBidShadeResponse{
		HasBid: out.HasBid, CampaignID: out.CampaignID, ClearingPriceMicro: out.ClearingPriceMicro,
		RecommendedBidMicro: out.RecommendedBidMicro, ShadeDeltaMicro: out.ShadeDeltaMicro,
		NoBidReason: out.NoBidReason, SecondPriceDeltaPct: out.SecondPriceDeltaPct,
	}, nil
}

func (s *Service) RunFloorOptimizer(ctx context.Context) (int, error) {
	return rtbadmin.RunFloorOptimizer(ctx, rtbAdminHost{svc: s})
}

func (s *Service) ApplyRtbFloorSuggestions(ctx context.Context, dryRun bool, placementIDs []string) (RtbFloorsApplyResult, error) {
	return rtbadmin.ApplyRtbFloorSuggestions(ctx, rtbAdminHost{svc: s}, dryRun, placementIDs)
}

func (s *Service) OptimizeBidFloors(ctx context.Context) ([]BidFloorRecommendationDTO, error) {
	return rtbadmin.OptimizeBidFloors(ctx, rtbAdminHost{svc: s})
}

func (s *Service) StartFloorOptimizerWorker(interval time.Duration) {
	if s == nil {
		return
	}
	w := rtbadmin.NewFloorOptimizerWorker(s, interval)
	s.StartBackgroundWorker(func() {
		w.Start(s.ctx)
	})
}

func (r rtbRuntimeConfig) RtbMode() string {
	if r.cfg == nil || r.cfg.RtbMode == "" {
		return "off"
	}
	return r.cfg.RtbMode
}

func (r rtbRuntimeConfig) RtbEnabled() bool {
	return r.cfg != nil && r.cfg.RtbEnabled()
}

func (r rtbRuntimeConfig) RtbExchangeNoBidMode() string {
	if r.cfg == nil {
		return ""
	}
	return r.cfg.RtbExchangeNoBidMode
}

var (
	_ rtbadmin.Host               = rtbAdminHost{}
	_ rtbadmin.FloorsHost         = rtbAdminHost{}
	_ rtbadmin.FloorOptimizerHost = (*Service)(nil)
)

func (s *Service) RtbReconcileCHStats(ctx context.Context, requestID string, window time.Duration) (reconciliation.RtbReconcileCHStats, bool) {
	return reconciliation.RTBCHStats(ctx, s, requestID, window)
}

var _ supply.Host = (*Service)(nil)

func (s *Service) SupplyStore() *supply.Store {
	if s == nil {
		return nil
	}
	if s.supplyStore == nil {
		s.supplyStore = supply.NewStore(s.pool, s)
	}
	return s.supplyStore
}

func (s *Service) ActorUserID(ctx context.Context) uuid.UUID {
	if u, ok := GetUser(ctx); ok {
		return u.UserID
	}
	return uuid.Nil
}

func (s *Service) ErrValidation(msg string) error {
	return errValidation(msg)
}

func (s *Service) EnqueueSupplyFilesUpdate(ctx context.Context, q db.Querier, trigger string) error {
	payload, err := coldpath.MarshalOutbox(SupplyFilesPayload{Trigger: trigger})
	if err != nil {
		return err
	}
	_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
		EventType: "UPDATE_SUPPLY_FILES",
		Payload:   payload,
	})
	return err
}

func (s *Service) SupplyExportPath() string {
	if s.cfg != nil && s.cfg.Management.SupplyExportPath != "" {
		return s.cfg.Management.SupplyExportPath
	}
	return "./data/supply-export"
}

func (s *Service) ListSellers(ctx context.Context) ([]SellerDTO, error) {
	return s.SupplyStore().ListSellers(ctx)
}

func (s *Service) GetSeller(ctx context.Context, id int64) (SellerDTO, error) {
	return s.SupplyStore().GetSeller(ctx, id)
}

func (s *Service) CreateSeller(ctx context.Context, spec SellerCreateSpec) (SellerDTO, error) {
	return s.SupplyStore().CreateSeller(ctx, spec)
}

func (s *Service) UpdateSeller(ctx context.Context, id int64, spec SellerUpdateSpec) (SellerDTO, error) {
	return s.SupplyStore().UpdateSeller(ctx, id, spec)
}

func (s *Service) DeleteSeller(ctx context.Context, id int64) error {
	return s.SupplyStore().DeleteSeller(ctx, id)
}

func (s *Service) ListAdsTxtEntries(ctx context.Context) ([]AdsTxtEntryDTO, error) {
	return s.SupplyStore().ListAdsTxtEntries(ctx)
}

func (s *Service) GetAdsTxtEntry(ctx context.Context, id int64) (AdsTxtEntryDTO, error) {
	return s.SupplyStore().GetAdsTxtEntry(ctx, id)
}

func (s *Service) CreateAdsTxtEntry(ctx context.Context, spec AdsTxtEntryCreateSpec) (AdsTxtEntryDTO, error) {
	return s.SupplyStore().CreateAdsTxtEntry(ctx, spec)
}

func (s *Service) UpdateAdsTxtEntry(ctx context.Context, id int64, spec AdsTxtEntryUpdateSpec) (AdsTxtEntryDTO, error) {
	return s.SupplyStore().UpdateAdsTxtEntry(ctx, id, spec)
}

func (s *Service) DeleteAdsTxtEntry(ctx context.Context, id int64) error {
	return s.SupplyStore().DeleteAdsTxtEntry(ctx, id)
}

func (s *Service) BuildSellersJSON(ctx context.Context) ([]byte, error) {
	return s.SupplyStore().BuildSellersJSON(ctx)
}

func (s *Service) GetSellersJSON(ctx context.Context) ([]byte, error) {
	return s.SupplyStore().GetSellersJSON(ctx)
}

func (s *Service) BuildAdsTxt(ctx context.Context) (string, error) {
	return s.SupplyStore().BuildAdsTxt(ctx)
}

type supplyAdminHost struct {
	svc *Service
}

var _ supply.AdminHost = supplyAdminHost{}

func (h supplyAdminHost) ListSellers(ctx context.Context) ([]supply.SellerDTO, error) {
	return h.svc.ListSellers(ctx)
}

func (h supplyAdminHost) CreateSeller(ctx context.Context, spec supply.SellerCreateSpec) (supply.SellerDTO, error) {
	return h.svc.CreateSeller(ctx, spec)
}

func (h supplyAdminHost) UpdateSeller(ctx context.Context, id int64, spec supply.SellerUpdateSpec) (supply.SellerDTO, error) {
	return h.svc.UpdateSeller(ctx, id, spec)
}

func (h supplyAdminHost) DeleteSeller(ctx context.Context, id int64) error {
	return h.svc.DeleteSeller(ctx, id)
}

func (h supplyAdminHost) ListAdsTxtEntries(ctx context.Context) ([]supply.AdsTxtEntryDTO, error) {
	return h.svc.ListAdsTxtEntries(ctx)
}

func (h supplyAdminHost) CreateAdsTxtEntry(ctx context.Context, spec supply.AdsTxtEntryCreateSpec) (supply.AdsTxtEntryDTO, error) {
	return h.svc.CreateAdsTxtEntry(ctx, spec)
}

func (h supplyAdminHost) UpdateAdsTxtEntry(ctx context.Context, id int64, spec supply.AdsTxtEntryUpdateSpec) (supply.AdsTxtEntryDTO, error) {
	return h.svc.UpdateAdsTxtEntry(ctx, id, spec)
}

func (h supplyAdminHost) DeleteAdsTxtEntry(ctx context.Context, id int64) error {
	return h.svc.DeleteAdsTxtEntry(ctx, id)
}

func (h supplyAdminHost) BuildSellersJSON(ctx context.Context) ([]byte, error) {
	return h.svc.BuildSellersJSON(ctx)
}

func (h supplyAdminHost) BuildAdsTxt(ctx context.Context) (string, error) {
	return h.svc.BuildAdsTxt(ctx)
}

func (h supplyAdminHost) SupplyExportPath() string {
	return h.svc.SupplyExportPath()
}

func (h supplyAdminHost) ValidateSupplyFiles(ctx context.Context) (supply.ValidationDTO, error) {
	report, err := h.svc.ValidateSupplyFiles(ctx)
	if err != nil {
		return supply.ValidationDTO{}, err
	}
	return supply.ValidationDTO{
		SellersJSONValid:      report.SellersJSONValid,
		SellersChecksumSHA256: report.SellersChecksumSHA256,
		SellersCount:          report.SellersCount,
		AdsTxtValid:           report.AdsTxtValid,
		AdsTxtChecksumSHA256:  report.AdsTxtChecksumSHA256,
		AdsTxtLineCount:       report.AdsTxtLineCount,
		Issues:                report.Issues,
	}, nil
}

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

func (h dashboardRoleHost) compositeReads() *CompositeReadService {
	return NewCompositeReadService(h.svc.GetPool(), h.svc.cfg)
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
	return listCustomerCampaignIDs(ctx, h.svc.GetPool(), customerID)
}

func (h dashboardReportHost) QueryWorstIVTSources(ctx context.Context, campaignIDs []uuid.UUID, from, to time.Time, limit int) ([]reports.SourceRowDTO, error) {
	return QueryWorstIVTSources(ctx, h.svc.clickhouseQuery, campaignIDs, from, to, limit)
}

func (h dashboardReportHost) QueryWorstIVTCountries(ctx context.Context, campaignIDs []uuid.UUID, from, to time.Time, limit int) ([]reports.FraudGeoHintDTO, error) {
	return QueryWorstIVTCountries(ctx, h.svc.clickhouseQuery, campaignIDs, from, to, limit)
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

func (s *Service) GetBuyerPortfolio(ctx context.Context, customerID uuid.UUID) (BuyerPortfolioDTO, error) {
	return s.buyerPortfolio().GetBuyerPortfolio(ctx, customerID)
}

func (s *Service) GetBuyerPortfolioRange(ctx context.Context, customerID uuid.UUID, from, to time.Time) (BuyerPortfolioDTO, error) {
	return s.buyerPortfolio().GetBuyerPortfolioRange(ctx, customerID, from, to)
}

func (s *Service) GetCampaignDashboard(ctx context.Context, campaignID uuid.UUID) (CampaignDashboardDTO, error) {
	return s.campaignDashboard().GetCampaignDashboard(ctx, campaignID)
}

func (s *Service) GetAdOpsDashboard(ctx context.Context, customerID uuid.UUID) (AdOpsDashboardDTO, error) {
	return s.roleDashboard().GetAdOpsDashboard(ctx, customerID)
}

func (s *Service) GetCFODashboard(ctx context.Context, customerID uuid.UUID) (CFODashboardDTO, error) {
	return s.roleDashboard().GetCFODashboard(ctx, customerID)
}

func (s *Service) GetAccountantDashboard(ctx context.Context, customerID uuid.UUID) (AccountantDashboardDTO, error) {
	return s.roleDashboard().GetAccountantDashboard(ctx, customerID)
}

func (s *Service) GetFraudDashboard(ctx context.Context, customerID uuid.UUID) (FraudDashboardDTO, error) {
	return s.roleDashboard().GetFraudDashboard(ctx, customerID)
}

func (s *Service) GetFraudDashboardRange(ctx context.Context, customerID uuid.UUID, from, to time.Time) (FraudDashboardDTO, error) {
	return s.roleDashboard().GetFraudDashboardRange(ctx, customerID, from, to)
}

func (s *Service) ResolvePublisherBind(ctx context.Context, userID uuid.UUID) (PublisherBind, error) {
	return s.dashboardPublisherService().ResolvePublisherBind(ctx, userID)
}

func (s *Service) GetPublisherDashboard(ctx context.Context, bind PublisherBind, from, to time.Time) (PublisherDashboardDTO, error) {
	return s.dashboardPublisherService().GetPublisherDashboard(ctx, bind, from, to)
}

func (s *Service) ListPublisherStatements(ctx context.Context, bind PublisherBind, from, to time.Time, limit, offset int32) ([]PublisherStatementDTO, int64, error) {
	return s.dashboardPublisherService().ListPublisherStatements(ctx, bind, from, to, limit, offset)
}

func (s *Service) ValidateSupplyFiles(ctx context.Context) (SupplyValidationReport, error) {
	report, err := s.dashboardPublisherService().ValidateSupplyFiles(ctx)
	if err != nil {
		return SupplyValidationReport{}, err
	}
	return report, nil
}

var (
	_ dashboardadmin.BuyerPortfolioReader    = (*Service)(nil)
	_ dashboardadmin.CampaignDashboardReader = (*Service)(nil)
	_ dashboardadmin.RoleDashboardReader     = (*Service)(nil)
	_ dashboardadmin.PublisherReader         = (*Service)(nil)
)

var (
	_ shardadmin.Host             = (*Service)(nil)
	_ shardadmin.HealthHost       = (*Service)(nil)
	_ shardadmin.OrchestratorHost = (*Service)(nil)
	_ shardadmin.LeaseHost        = (*Service)(nil)
	_ shardadmin.CatchupHost      = (*Service)(nil)
)

func (s *Service) AlertSlotMapMigrating(ctx context.Context, version int32, slots []int16, targetShard int16) {
	if s != nil && s.alerter != nil {
		s.alerter.AlertSlotMapMigrating(ctx, version, slots, targetShard)
	}
}

func (s *Service) AlertSlotMigrationError(ctx context.Context, stage string, err error) {
	if s != nil && s.alerter != nil {
		s.alerter.AlertSlotMigrationError(ctx, stage, err)
	}
}

func (s *Service) ListActiveCampaignUUIDs(ctx context.Context) ([]uuid.UUID, error) {
	rows, err := db.New(s.GetPool()).ListCampaignIDs(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]uuid.UUID, 0, len(rows))
	for _, row := range rows {
		if !row.Valid {
			continue
		}
		out = append(out, uuid.UUID(row.Bytes))
	}
	return out, nil
}

func (s *Service) SlotMigrationDualWriteEnabled() bool {
	return s != nil && s.cfg != nil && s.cfg.SlotMigrationDualWriteEnabled
}

func (s *Service) DualWriteConfig() domain.SlotMigrationDualWriteConfig {
	cfg := domain.SlotMigrationDualWriteConfig{
		Enabled:      s.SlotMigrationDualWriteEnabled(),
		LagEpsilon:   0,
		LagThreshold: 1000,
	}
	if s == nil || s.cfg == nil {
		return cfg
	}
	cfg.LagEpsilon = s.cfg.SlotMigrationLagEpsilon
	cfg.LagThreshold = s.cfg.SlotMigrationLagThreshold
	if cfg.LagThreshold <= 0 {
		cfg.LagThreshold = 1000
	}
	return cfg
}

func (s *Service) MigrationFenceEnabled() bool {
	return s != nil && s.cfg != nil && s.cfg.MigrationFenceEnabled
}

func (s *Service) AfterSlotMapActivated(ctx context.Context, version int32) {
	s.afterSlotMapActivated(ctx, version)
}

func (s *Service) GetSlotMap(ctx context.Context, version *int32, includeSlots bool) (SlotMapVersionDTO, error) {
	return shardadmin.GetSlotMap(ctx, s.GetPool(), version, includeSlots)
}

func (s *Service) CreateSlotMapVersion(ctx context.Context, adminID uuid.UUID, baseVersion *int32, overrides []domain.SlotOverride) (int32, error) {
	return shardadmin.CreateSlotMapVersion(ctx, s, adminID, baseVersion, overrides)
}

func (s *Service) MarkSlotMapMigrating(ctx context.Context, adminID uuid.UUID, version int32, slots []int16, targetShard int16) error {
	return shardadmin.MarkSlotMapMigrating(ctx, s, adminID, version, slots, targetShard)
}

func (s *Service) ActivateSlotMapVersion(ctx context.Context, adminID uuid.UUID, version int32) error {
	return shardadmin.ActivateSlotMapVersionWithMigration(ctx, s, adminID, version)
}

func (s *Service) ActivateSlotMapVersionWithMigration(ctx context.Context, adminID uuid.UUID, version int32) error {
	return shardadmin.ActivateSlotMapVersionWithMigration(ctx, s, adminID, version)
}

func (s *Service) GetSlotMigrations(ctx context.Context, version int32) ([]SlotMigrationDTO, error) {
	return shardadmin.GetSlotMigrations(ctx, s.GetPool(), version)
}

func (s *Service) EnsureSlotMigrationJobs(ctx context.Context, draftVersion int32) error {
	return shardadmin.EnsureSlotMigrationJobs(ctx, s, draftVersion)
}

func (s *Service) CopySlotMigrationData(ctx context.Context, version int32, slot int16) error {
	return shardadmin.CopySlotMigrationData(ctx, s, version, slot)
}

func (s *Service) CopyAllMigratingSlots(ctx context.Context, draftVersion int32) error {
	return shardadmin.CopyAllMigratingSlots(ctx, s, draftVersion)
}

func (s *Service) DrainMigratingSlots(ctx context.Context, version int32) error {
	return shardadmin.DrainMigratingSlots(ctx, s, version)
}

func (s *Service) RollbackSlotMapVersion(ctx context.Context, adminID uuid.UUID, previousVersion int32) error {
	return shardadmin.RollbackSlotMapVersion(ctx, s, adminID, previousVersion)
}

func (s *Service) CatchUpDualWriteSlots(ctx context.Context, draftVersion int32) error {
	return shardadmin.CatchUpDualWriteSlots(ctx, s, draftVersion)
}

func (s *Service) VerifySlotMigrationR5(ctx context.Context) error {
	return shardadmin.VerifySlotMigrationR5(ctx, s)
}

func (s *Service) HasPendingSlotDrain(ctx context.Context) (bool, error) {
	return shardadmin.HasPendingSlotDrain(ctx, s.GetPool())
}

func (s *Service) BumpFencesForPendingMigrations(ctx context.Context) error {
	return shardadmin.BumpFencesForPendingMigrations(ctx, s)
}

func (s *Service) PublishRtbCatalogReload(ctx context.Context) error {
	return shardadmin.PublishControlChannelToAllShards(ctx, s.redisShards, domain.RtbCatalogReloadChannel(s.cfg), "reload")
}

func (s *Service) DedupAdapter(ctx context.Context) *dedup.Adapter {
	if s == nil || s.pool == nil || s.cfg == nil {
		return nil
	}
	return shardadmin.DedupAdapter(ctx, s.pool, s.cfg.RegionCode)
}

func (s *Service) OperationLeaseWorker() *OperationLeaseWorker {
	if s == nil {
		return nil
	}
	return s.leaseWorker
}

func NewOperationLeaseWorker(svc *Service) *OperationLeaseWorker {
	return shardadmin.NewOperationLeaseWorker(svc)
}

func (s *Service) ControlRedis() redis.UniversalClient {
	return shardadmin.PickHealthyControlShard(s.redisShards)
}

func (s *Service) LeaseWorkerConfig() shardadmin.LeaseWorkerConfig {
	cfg := shardadmin.LeaseWorkerConfig{NodeRole: "management"}
	if s == nil || s.cfg == nil {
		return cfg
	}
	cfg.NodeID = s.cfg.NodeID
	if s.cfg.NodeRole != "" {
		cfg.NodeRole = s.cfg.NodeRole
	}
	cfg.RegionCode = int16(s.cfg.RegionCode)
	cfg.OpLeaseTimeoutSec = s.cfg.OpLeaseTimeoutSec
	cfg.OpLeaseMaxRenewals = int32(s.cfg.OpLeaseMaxRenewals)
	cfg.OpLeaseFencingDir = s.cfg.OpLeaseFencingDir
	return cfg
}

func (s *Service) LeaseReplicaNodes() []string {
	if s != nil && s.cfg != nil && s.cfg.NodeID != "" {
		return []string{s.cfg.NodeID}
	}
	return []string{"management"}
}

func (s *Service) GetShardHealth(ctx context.Context) (shardadmin.ShardHealthReport, error) {
	return shardadmin.GetShardHealth(ctx, s)
}

func (s *Service) PublishRoutingCutover(ctx context.Context, routingEpoch int64, slotVersion int32) {
	s.publishRoutingCutover(ctx, routingEpoch, slotVersion)
}

func (s *Service) AutoscaleShards(ctx context.Context, provider shardadmin.ShardMetricsProvider, cfg shardadmin.ShardAutoscaleConfig) (int32, error) {
	return shardadmin.AutoscaleShards(ctx, s, provider, cfg)
}

func NewShardOrchestrator(svc *Service, provider shardadmin.ShardMetricsProvider, interval time.Duration) *shardadmin.ShardOrchestrator {
	return shardadmin.NewShardOrchestrator(svc, provider, interval)
}

func NewShard0CatchupWorker(svc *Service, redisOpts database.RedisShardOptions) *shardadmin.Shard0CatchupWorker {
	return shardadmin.NewShard0CatchupWorker(svc, redisOpts)
}

func (s *Service) TryReconnectShard0(ctx context.Context, opts database.RedisShardOptions) bool {
	if s == nil || s.cfg == nil {
		return false
	}
	s.shard0Mu.Lock()
	defer s.shard0Mu.Unlock()

	if len(s.redisShards) == 0 || s.redisShards[0] != nil {
		return false
	}
	redisClient, err := database.ConnectRedisShard(ctx, s.cfg, 0, opts)
	if err != nil {
		return false
	}
	s.redisShards[0] = redisClient
	database.SetShard0ClientNilMetric(s.redisShards)
	return true
}

func (s *Service) RunShard0Catchup(ctx context.Context) error {
	return shardadmin.RunShard0Catchup(ctx, s)
}

func NewSlotMigrationOrchestrator(svc *Service, interval time.Duration) *shardadmin.SlotMigrationOrchestrator {
	return shardadmin.NewSlotMigrationOrchestrator(svc, interval)
}

func (s *Service) afterSlotMapActivated(ctx context.Context, version int32) {
	routingEpoch := int64(0)
	if row, err := domain.NewCampaignRoutingRepo(s.GetPool()).BumpGlobalRoutingEpoch(ctx); err == nil {
		routingEpoch = row.RoutingEpoch
		version = row.ActiveVersion
	}
	if ss, ok := s.sharder.(*domain.StaticSlotSharder); ok {
		_, _ = domain.LoadActiveSlotMap(ctx, s.GetPool(), ss, len(s.redisShards))
	}
	s.publishRoutingCutover(ctx, routingEpoch, version)
}

var (
	_ nodeadmin.ScorerHost  = (*Service)(nil)
	_ nodeadmin.MetricsHost = (*Service)(nil)
)

func (s *Service) ScorerConfig() nodeadmin.ScorerConfig {
	if s == nil || s.cfg == nil {
		return nodeadmin.DefaultScorerConfig()
	}
	return nodeadmin.ScorerConfigFrom(s.cfg)
}

func (s *Service) RegionCode() int16 {
	if s == nil || s.cfg == nil {
		return 0
	}
	return int16(s.cfg.RegionCode)
}

func (s *Service) MultiRegionGlobal() bool {
	return s != nil && s.cfg != nil && s.cfg.MultiRegionGlobal()
}

func (s *Service) UDPSyncInterval() time.Duration {
	if s == nil || s.cfg == nil || s.cfg.UDPSyncIntervalMs <= 0 {
		return 0
	}
	return time.Duration(s.cfg.UDPSyncIntervalMs) * time.Millisecond
}

func (s *Service) ScoringMetricsForRole(role string) []nodeadmin.ScoringMetricDef {
	if s != nil && s.scoringWeights != nil {
		return s.scoringWeights.MetricsForRole(role)
	}
	return nodeadmin.MetricsForRole(role)
}

func (s *Service) NodeIdentity() (nodeID, role string, region int16) {
	nodeID, _ = os.Hostname()
	role = "management"
	if s != nil && s.cfg != nil {
		if s.cfg.NodeID != "" {
			nodeID = s.cfg.NodeID
		}
		if s.cfg.NodeRole != "" {
			role = s.cfg.NodeRole
		}
		region = int16(s.cfg.RegionCode)
	}
	return nodeID, role, region
}

func NewNodeCapacityScorer(svc *Service) *nodeadmin.NodeCapacityScorer {
	return nodeadmin.NewNodeCapacityScorer(svc)
}

func NewGlobalRegionTrafficScorer(svc *Service) *nodeadmin.GlobalRegionTrafficScorer {
	return nodeadmin.NewGlobalRegionTrafficScorer(svc)
}

func NewNodeCapacityScorerWorker(svc *Service) *nodeadmin.CapacityScorerWorker {
	return nodeadmin.NewCapacityScorerWorker(svc)
}

func NewGlobalRegionTrafficScorerWorker(svc *Service) *nodeadmin.GlobalTrafficScorerWorker {
	return nodeadmin.NewGlobalTrafficScorerWorker(svc)
}

func NewNodeMetricsWorker(svc *Service) *nodeadmin.MetricsWorker {
	return nodeadmin.NewMetricsWorker(svc)
}

func NewNodeMetricsSnapshotWorker(svc *Service) *nodeadmin.MetricsSnapshotWorker {
	return nodeadmin.NewMetricsSnapshotWorker(svc)
}

func NewScoringWeightsStore(ctx context.Context, pool *pgxpool.Pool, cfg *config.Config) (*nodeadmin.ScoringWeightsStore, error) {
	return nodeadmin.NewScoringWeightsStore(ctx, pool, cfg)
}

func ValidateScoringWeightsConfig(ctx context.Context, pool *pgxpool.Pool, cfg *config.Config) error {
	return nodeadmin.ValidateScoringWeightsConfig(ctx, pool, cfg)
}

type (
	ScorerConfig              = nodeadmin.ScorerConfig
	ScoringMetricDef          = nodeadmin.ScoringMetricDef
	ScoringWeightsStore       = nodeadmin.ScoringWeightsStore
	NodeCapacityScorer        = nodeadmin.NodeCapacityScorer
	GlobalRegionTrafficScorer = nodeadmin.GlobalRegionTrafficScorer
	NodeScoreInput            = nodeadmin.NodeScoreInput
	NodeScoreResult           = nodeadmin.NodeScoreResult
	NodeScoreState            = nodeadmin.NodeScoreState
	BucketPoint               = nodeadmin.BucketPoint
	RegionDialInput           = nodeadmin.RegionDialInput
	RegionDialResult          = nodeadmin.RegionDialResult
	MetricKind                = nodeadmin.MetricKind
	ScorePhase                = nodeadmin.ScorePhase
)

const (
	RoleTracker     = nodeadmin.RoleTracker
	RoleRegionProxy = nodeadmin.RoleRegionProxy
	RoleProcessor   = nodeadmin.RoleProcessor

	ProvenanceOwnWindow           = nodeadmin.ProvenanceOwnWindow
	ProvenanceNeighborMedian      = nodeadmin.ProvenanceNeighborMedian
	ProvenanceHistoricalDaily     = nodeadmin.ProvenanceHistoricalDaily
	ProvenanceConservativeDefault = nodeadmin.ProvenanceConservativeDefault

	MetricLatency     = nodeadmin.MetricLatency
	MetricUtilization = nodeadmin.MetricUtilization
	MetricRate        = nodeadmin.MetricRate
	MetricCounter     = nodeadmin.MetricCounter
)

func DefaultScorerConfig() ScorerConfig { return nodeadmin.DefaultScorerConfig() }

func ScorerConfigFrom(cfg *config.Config) ScorerConfig { return nodeadmin.ScorerConfigFrom(cfg) }

func ScoreNode(in NodeScoreInput, cfg ScorerConfig) NodeScoreResult {
	return nodeadmin.ScoreNode(in, cfg)
}

func ScoreNodes(inputs []NodeScoreInput, cfg ScorerConfig) []NodeScoreResult {
	return nodeadmin.ScoreNodes(inputs, cfg)
}

func ComputeCapacityScoreFromValues(role string, values map[string]float64, defs []ScoringMetricDef) float64 {
	return nodeadmin.ComputeCapacityScoreFromValues(role, values, defs)
}

func MetricsForRole(role string) []ScoringMetricDef { return nodeadmin.MetricsForRole(role) }

func DefaultTrackerMetrics() []ScoringMetricDef { return nodeadmin.DefaultTrackerMetrics() }

func DefaultRegionProxyMetrics() []ScoringMetricDef { return nodeadmin.DefaultRegionProxyMetrics() }

func ComputeRegionDialResults(inputs []RegionDialInput, cfg ScorerConfig) []RegionDialResult {
	return nodeadmin.ComputeRegionDialResults(inputs, cfg)
}

func NormalizePeerWeights(weights []float64, minW, maxW float64) []float64 {
	return nodeadmin.NormalizePeerWeights(weights, minW, maxW)
}

func ApplyHardSignals(weight float64, diskDegraded, budgetInvariantFail bool) float64 {
	return nodeadmin.ApplyHardSignals(weight, diskDegraded, budgetInvariantFail)
}

func SortNodeIDs(ids []string) []string { return nodeadmin.SortNodeIDs(ids) }

func HistoricalSnapshotDay(now time.Time) time.Time { return nodeadmin.HistoricalSnapshotDay(now) }

func LookupHistoricalDaily(ctx context.Context, pool *pgxpool.Pool, region int16, role, metric string, kind MetricKind, now time.Time) (*float64, error) {
	return nodeadmin.LookupHistoricalDaily(ctx, pool, region, role, metric, kind, now)
}
