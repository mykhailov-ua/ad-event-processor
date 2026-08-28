package controlplane

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"ad-event-processor/internal/automation"
	"ad-event-processor/internal/brand"
	"ad-event-processor/internal/campaign"
	"ad-event-processor/internal/database"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/ledger"
	"ad-event-processor/internal/marginguard"
	"ad-event-processor/internal/platformadmin"
	"ad-event-processor/internal/smartalerts"
	"ad-event-processor/pkg/coldpath"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	_ brand.Host       = (*Service)(nil)
	_ marginguard.Host = (*Service)(nil)
)

func (s *Service) StartAutomationWorker(ctx context.Context, intervalMinutes int) {
	if s == nil || s.cfg == nil || !s.cfg.Management.AutomationRulesEnabled {
		return
	}
	clickhouseQuery := s.ClickHouseQuery()
	if clickhouseQuery == nil {
		return
	}
	interval := time.Duration(intervalMinutes) * time.Minute
	maxEvals := 50
	if s.cfg.Management.AutomationRulesMaxEvalsPerCustomerPerTick > 0 {
		maxEvals = s.cfg.Management.AutomationRulesMaxEvalsPerCustomerPerTick
	}
	host := automationHost{s}
	w := automation.NewWorker(s.pool, clickhouseQuery, automation.NewExecutor(host), interval, maxEvals)
	s.StartBackgroundWorker(func() {
		w.Start(ctx)
	})
	slog.Info("automation rules worker enabled", "interval", interval)
}

type automationHost struct {
	svc *Service
}

func (h automationHost) Pool() *pgxpool.Pool {
	return h.svc.GetPool()
}

func (h automationHost) PauseCampaign(ctx context.Context, campaignID uuid.UUID, reason string) error {
	return h.svc.PauseCampaign(ctx, campaignID, reason)
}

func (h automationHost) BlacklistPlacement(ctx context.Context, campaignID uuid.UUID, placementID string) error {
	return h.svc.BlockCampaignPlacement(ctx, campaignID, placementID)
}

type (
	SmartAlertsHTTPHandlers     = smartalerts.HTTPHandlers
	SmartAlertRuleDTO           = smartalerts.SmartAlertRuleDTO
	SmartAlertEventDTO          = smartalerts.SmartAlertEventDTO
	UpsertSmartAlertRuleRequest = smartalerts.UpsertSmartAlertRuleRequest
)

func (s *Service) SmartAlertsStore() *smartalerts.Store {
	if s == nil {
		return nil
	}
	return smartalerts.NewStore(smartalertsHost{s})
}

func (s *Service) ListSmartAlertRules(ctx context.Context, customerID uuid.UUID) ([]SmartAlertRuleDTO, error) {
	return s.SmartAlertsStore().ListSmartAlertRules(ctx, customerID)
}

func (s *Service) CreateSmartAlertRule(ctx context.Context, req UpsertSmartAlertRuleRequest) (SmartAlertRuleDTO, error) {
	return s.SmartAlertsStore().CreateSmartAlertRule(ctx, req)
}

func (s *Service) UpdateSmartAlertRule(ctx context.Context, ruleID uuid.UUID, req UpsertSmartAlertRuleRequest) (SmartAlertRuleDTO, error) {
	return s.SmartAlertsStore().UpdateSmartAlertRule(ctx, ruleID, req)
}

func (s *Service) DeleteSmartAlertRule(ctx context.Context, ruleID uuid.UUID) error {
	return s.SmartAlertsStore().DeleteSmartAlertRule(ctx, ruleID)
}

func (s *Service) ListSmartAlertHistory(ctx context.Context, customerID uuid.UUID, limit int) ([]SmartAlertEventDTO, error) {
	return s.SmartAlertsStore().ListSmartAlertHistory(ctx, customerID, limit)
}

func (s *Service) AckSmartAlertEvent(ctx context.Context, eventID, actorID uuid.UUID) error {
	return s.SmartAlertsStore().AckSmartAlertEvent(ctx, eventID, actorID)
}

func (s *Service) StartSmartAlertsWorker(ctx context.Context, interval time.Duration) {
	if s == nil || s.cfg == nil || !s.cfg.Management.SmartAlertsEnabled {
		return
	}
	store := s.SmartAlertsStore()
	w := smartalerts.NewWorker(smartalertsHost{s}, store, interval)
	s.StartBackgroundWorker(func() {
		w.Start(ctx)
	})
	slog.Info("smart alerts worker enabled", "interval", interval)
}

func (s *Service) CheckStuckDrainJobs(ctx context.Context) {
	smartalerts.CheckStuckDrainJobs(ctx, smartalertsHost{s})
}

type smartalertsHost struct {
	svc *Service
}

func (h smartalertsHost) Pool() *pgxpool.Pool {
	return h.svc.GetPool()
}

func (h smartalertsHost) ClickHouseQuery() *database.ClickHouseQuery {
	return h.svc.ClickHouseQuery()
}

func (h smartalertsHost) DrainStuckThresholdSec() int {
	if h.svc.cfg == nil {
		return 0
	}
	return h.svc.cfg.Management.DrainStuckThresholdSec
}

func (h smartalertsHost) AlertDrainStuck(ctx context.Context, version int32, slot int16, state, lastError string, updatedAt time.Time) {
	if h.svc.alerter != nil {
		h.svc.alerter.AlertDrainStuck(ctx, version, slot, state, lastError, updatedAt)
	}
}

func (s *Service) BrandStore() *brand.Store {
	if s == nil {
		return nil
	}
	if s.brandStore == nil {
		s.brandStore = brand.NewStore(s.pool, s)
	}
	return s.brandStore
}

func (s *Service) ErrBrandNotFound() error        { return ErrBrandNotFound }
func (s *Service) ErrCreativeNotFound() error     { return ErrCreativeNotFound }
func (s *Service) ErrWeightMustBePositive() error { return ErrWeightMustBePositive }
func (s *Service) ErrCreativeStatusInvalid() error {
	return ErrCreativeStatusInvalid
}

func (s *Service) MapNotFound(err, target error) error {
	return mapNotFound(err, target)
}

func (s *Service) OnConfigureBrandFcap(ctx context.Context, q db.Querier, brandID uuid.UUID, prev db.AdvertiserBrand, limit, window int32) error {
	payloadBytes, err := coldpath.MarshalOutbox(brandFcapOutboxPayload{
		BrandID:    brandID.String(),
		FreqLimit:  limit,
		FreqWindow: window,
	})
	if err != nil {
		return fmt.Errorf("marshal configure brand fcap outbox payload: %w", err)
	}
	if _, err := q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
		EventType: "CONFIGURE_BRAND_FCAP",
		Payload:   payloadBytes,
	}); err != nil {
		return fmt.Errorf("failed to create outbox event: %w", err)
	}
	s.AuditLog(ctx, q, uuid.Nil, "CONFIGURE_BRAND_FCAP", "brand", &brandID, platformadmin.AuditBrandFcapChange{
		OldFreqLimit:  prev.FreqLimit,
		OldFreqWindow: prev.FreqWindow,
		NewFreqLimit:  limit,
		NewFreqWindow: window,
	}, nil)
	return nil
}

func (s *Service) OnBrandCreativesChanged(ctx context.Context, q db.Querier, brandID uuid.UUID) error {
	return s.emitBrandCreativesOutbox(ctx, q, brandID)
}

func (s *Service) CreateBrand(ctx context.Context, customerID uuid.UUID, name string) (uuid.UUID, error) {
	return s.BrandStore().CreateBrand(ctx, customerID, name)
}

func (s *Service) GetBrandDTO(ctx context.Context, id uuid.UUID) (brand.DTO, error) {
	return s.BrandStore().GetBrandDTO(ctx, id)
}

func (s *Service) ListBrandsByCustomer(ctx context.Context, customerID uuid.UUID) ([]brand.DTO, error) {
	return s.BrandStore().ListBrandsByCustomer(ctx, customerID)
}

func (s *Service) ConfigureBrandFcap(ctx context.Context, brandID uuid.UUID, limit, window int32) error {
	return s.BrandStore().ConfigureBrandFcap(ctx, brandID, limit, window)
}

func (s *Service) UpsertBrandCreative(ctx context.Context, brandID uuid.UUID, name, landingURL string, weight int32, status string) (uuid.UUID, error) {
	return s.BrandStore().UpsertBrandCreative(ctx, brandID, name, landingURL, weight, status)
}

func (s *Service) ListBrandCreatives(ctx context.Context, brandID uuid.UUID) ([]brand.CreativeDTO, error) {
	return s.BrandStore().ListBrandCreatives(ctx, brandID)
}

func (s *Service) UpdateBrandCreative(ctx context.Context, creativeID uuid.UUID, name, landingURL string, weight int32, status string) error {
	return s.BrandStore().UpdateBrandCreative(ctx, creativeID, name, landingURL, weight, status)
}

func (s *Service) DeleteBrandCreative(ctx context.Context, creativeID uuid.UUID) error {
	return s.BrandStore().DeleteBrandCreative(ctx, creativeID)
}

func (s *Service) emitBrandCreativesOutbox(ctx context.Context, q db.Querier, brandID uuid.UUID) error {
	payload, err := coldpath.MarshalOutbox(brandIDPayload{BrandID: brandID.String()})
	if err != nil {
		return fmt.Errorf("marshal sync brand creatives outbox payload: %w", err)
	}
	_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{EventType: "SYNC_BRAND_CREATIVES", Payload: payload})
	return err
}

func (s *Service) MarginGuardStore() *marginguard.Store {
	if s == nil {
		return nil
	}
	if s.marginGuardStore == nil {
		s.marginGuardStore = marginguard.NewStore(s.pool, s)
	}
	return s.marginGuardStore
}

func (s *Service) DefaultCostOverRevenueThresholdBps() int {
	if s.cfg != nil && s.cfg.MarginGuardDefaultThresholdBps > 0 {
		return s.cfg.MarginGuardDefaultThresholdBps
	}
	return 500
}

func (s *Service) CreateMarginGuardPolicy(ctx context.Context, p *ledger.Policy) error {
	return s.MarginGuardStore().CreatePolicy(ctx, p)
}

func (s *Service) ListMarginGuardPolicies(ctx context.Context, campaignID uuid.UUID) ([]*ledger.Policy, error) {
	return s.MarginGuardStore().ListPolicies(ctx, campaignID)
}

func (s *Service) GetCampaignMargin(ctx context.Context, campaignID uuid.UUID) (campaign.CampaignMarginDTO, error) {
	m, err := s.MarginGuardStore().GetCampaignMargin(ctx, campaignID)
	if err != nil {
		return campaign.CampaignMarginDTO{}, err
	}
	return campaignMarginToDTO(m), nil
}

func (s *Service) AttachCampaignListMarginBreach(ctx context.Context, items []campaign.CampaignDTO) {
	s.MarginGuardStore().AttachCampaignListMarginBreach(ctx, items)
}

func (s *Service) GetMarginGuardActivity(ctx context.Context, campaignID uuid.UUID) ([]marginguard.ActivityRow, error) {
	rows, err := s.MarginGuardStore().ListActivity(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	out := make([]marginguard.ActivityRow, len(rows))
	for i, row := range rows {
		out[i] = marginguard.ActivityRow(row)
	}
	return out, nil
}

func (s *Service) RemovePlacementOverride(ctx context.Context, campaignID uuid.UUID, placementID string) error {
	return s.MarginGuardStore().RemovePlacementOverride(ctx, campaignID, placementID)
}

func (s *Service) BlockCampaignPlacement(ctx context.Context, campaignID uuid.UUID, placementID string) error {
	return s.MarginGuardStore().BlockCampaignPlacement(ctx, campaignID, placementID)
}

func (s *Service) batchCampaignMarginBreach(ctx context.Context, campaignIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
	return s.MarginGuardStore().BatchMarginBreach(ctx, campaignIDs)
}

func campaignMarginToDTO(m marginguard.CampaignMargin) campaign.CampaignMarginDTO {
	return campaign.CampaignMarginDTO{
		CampaignID:           m.CampaignID,
		WindowStart:          m.WindowStart,
		WindowHours:          m.WindowHours,
		AdvertiserSpendMicro: m.AdvertiserSpendMicro,
		RtbCostMicro:         m.RtbCostMicro,
		OperatorMarginMicro:  m.OperatorMarginMicro,
		PublisherPayoutMicro: m.PublisherPayoutMicro,
		CostOverRevenueLimit: m.CostOverRevenueLimit,
		ThresholdBps:         m.ThresholdBps,
		MarginBreach:         m.MarginBreach,
	}
}

type marginGuardServiceAdapter struct {
	svc *Service
}

func (a marginGuardServiceAdapter) ListPolicies(ctx context.Context, campaignID uuid.UUID) ([]*ledger.Policy, error) {
	return a.svc.ListMarginGuardPolicies(ctx, campaignID)
}

func (a marginGuardServiceAdapter) CreatePolicy(ctx context.Context, p *ledger.Policy) error {
	return a.svc.CreateMarginGuardPolicy(ctx, p)
}

func (a marginGuardServiceAdapter) ListActivity(ctx context.Context, campaignID uuid.UUID) ([]marginguard.ActivityRow, error) {
	rows, err := a.svc.GetMarginGuardActivity(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	out := make([]marginguard.ActivityRow, len(rows))
	for i, row := range rows {
		out[i] = marginguard.ActivityRow(row)
	}
	return out, nil
}

func (a marginGuardServiceAdapter) RemovePlacementOverride(ctx context.Context, campaignID uuid.UUID, placementID string) error {
	return a.svc.RemovePlacementOverride(ctx, campaignID, placementID)
}
