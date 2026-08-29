package controlplane

import (
	"context"
	"time"

	"ad-event-processor/internal/automation"

	"ad-event-processor/internal/campaign"
	"ad-event-processor/internal/campaign/wizard"
	campaignworker "ad-event-processor/internal/campaign/worker"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/platformadmin"
	"ad-event-processor/internal/reports"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func (s *Service) CampaignWorker() *campaignworker.Worker {
	if s == nil {
		return nil
	}
	if s.campaignWorker == nil {
		s.campaignWorker = campaignworker.NewWorker(s.CampaignRuntime(), s)
	}
	return s.campaignWorker
}

func (s *Service) MABInterval() time.Duration {
	if s == nil || s.cfg == nil || s.cfg.MABIntervalMs <= 0 {
		return 0
	}
	return time.Duration(s.cfg.MABIntervalMs) * time.Millisecond
}

func (s *Service) RunWithPostgresLow(ctx context.Context, fn func(context.Context) error) error {
	return s.withPostgresLow(ctx, fn)
}

func (s *Service) Pool() *pgxpool.Pool {
	return s.GetPool()
}

func (s *Service) PacingToleranceMargin() float64 {
	if s == nil || s.cfg == nil {
		return 0
	}
	return s.cfg.PacingToleranceMargin
}

func (s *Service) CampaignLocation(timezone string) *time.Location {
	return campaign.CampaignLocation(&s.timezoneLocationCache, timezone)
}

func (s *Service) PacingHourWeights(ctx context.Context) [24]float64 {
	return s.fetchPacingHourWeights(ctx)
}

func (s *Service) AuditPacingLoopAdjustment(ctx context.Context, q db.Querier, campaignID uuid.UUID, oldPacing, newPacing, spend, expected string) {
	s.AuditLog(ctx, q, uuid.Nil, "PACING_LOOP_ADJUSTMENT", "campaign", &campaignID, platformadmin.AuditPacingLoopAdjustment{
		OldPacing: oldPacing,
		NewPacing: newPacing,
		Spend:     spend,
		Expected:  expected,
		Curve:     "daypart_weighted",
	}, nil)
}

func (s *Service) EmitBrandCreativesOutbox(ctx context.Context, q db.Querier, brandID uuid.UUID) error {
	return s.emitBrandCreativesOutbox(ctx, q, brandID)
}

func (s *Service) CreateCampaignTemplate(ctx context.Context, customerID uuid.UUID, name string, budgetLimit int64, pacing db.PacingModeType, dailyBudget int64, timezone string, freqLimit, freqWindow int32, targetCountries []string, brandID *uuid.UUID, daypartHours []int16) (uuid.UUID, error) {
	return s.CampaignRuntime().CreateCampaignTemplate(ctx, customerID, name, budgetLimit, pacing, dailyBudget, timezone, freqLimit, freqWindow, targetCountries, brandID, daypartHours)
}

func (s *Service) ListCampaignTemplates(ctx context.Context, customerID uuid.UUID, limit, offset int32) ([]campaign.CampaignTemplateDTO, int64, error) {
	return s.CampaignRuntime().ListCampaignTemplates(ctx, customerID, limit, offset)
}

func (s *Service) CreateCampaignFromTemplate(ctx context.Context, templateID uuid.UUID, customerID uuid.UUID, name string, budgetLimit *int64, idempotencyKey string) (uuid.UUID, error) {
	return s.CampaignRuntime().CreateCampaignFromTemplate(ctx, templateID, customerID, name, budgetLimit, idempotencyKey)
}

func (s *Service) SaveCampaignAsTemplate(ctx context.Context, campaignID uuid.UUID, templateName string) (uuid.UUID, error) {
	return s.CampaignRuntime().SaveCampaignAsTemplate(ctx, campaignID, templateName)
}

func (s *Service) ProcessScheduleTick(ctx context.Context) error {
	return s.CampaignWorker().ProcessScheduleTick(ctx)
}

func (s *Service) RunDeliveryOptimizerTick(ctx context.Context, syncWorkers []*domain.SyncWorker, runMAB bool) error {
	return s.CampaignWorker().RunDeliveryOptimizerTick(ctx, syncWorkers, runMAB)
}

func (s *Service) ClosedLoopPacingController(ctx context.Context, syncWorkers []*domain.SyncWorker) error {
	return s.CampaignWorker().ClosedLoopPacingController(ctx, syncWorkers)
}

func (s *Service) RunVPPPacingController(ctx context.Context) error {
	return campaignworker.RunVPPPacingController(ctx, s)
}

func (s *Service) CampaignShard(campaignID uuid.UUID) int {
	if s == nil || s.sharder == nil {
		return 0
	}
	return s.sharder.GetShard(campaignID)
}

func (s *Service) QueryVPPCampaignSamplesBatch(ctx context.Context, from, to time.Time, campaignIDs []uuid.UUID) (map[uuid.UUID][]campaignworker.VPPCampaignSample, error) {
	return campaignworker.QueryVPPCampaignSamplesBatch(ctx, s.clickhouseQuery, from, to, campaignIDs)
}

func (s *Service) DrainWaitTimeoutMs() int64 {
	if s == nil || s.cfg == nil || s.cfg.Lifecycle.WaitTimeoutMs <= 0 {
		return 100
	}
	return int64(s.cfg.Lifecycle.WaitTimeoutMs)
}

func (s *Service) FinalizeDrainingCampaign(ctx context.Context, q db.Querier, campaignID uuid.UUID, camp db.Campaign, reason string) error {
	feePercent := 0.0
	if s.cfg != nil {
		feePercent = s.cfg.Management.CancellationFeePercent
	}
	return campaign.FinalizeDrainingCampaign(ctx, q, s, feePercent, campaignID, camp, reason)
}

func (s *Service) fetchPacingHourWeights(ctx context.Context) [24]float64 {
	if s.clickhouseQuery == nil {
		return campaign.UniformHourWeights()
	}
	lookbackEnd := time.Now().UTC().Truncate(time.Hour)
	lookbackStart := lookbackEnd.Add(-campaign.PacingLookbackDays * 24 * time.Hour)
	_, samples, err := reports.QueryForecastHourlySamples(ctx, forecastHost{svc: s}, lookbackStart, lookbackEnd, nil)
	if err != nil {
		return campaign.UniformHourWeights()
	}
	return reports.BuildHourWeights(samples)
}

var (
	_ campaignworker.DeliveryPostgresHost  = (*Service)(nil)
	_ campaignworker.PacingDeliveryHost    = (*Service)(nil)
	_ campaignworker.AutoscaleDeliveryHost = (*Service)(nil)
	_ campaignworker.BanditDeliveryHost    = (*Service)(nil)
	_ campaignworker.DeliveryPublishHost   = (*Service)(nil)
	_ campaignworker.DeliveryHost          = (*Service)(nil)
	_ campaign.Effects                     = (*Service)(nil)
	_ campaign.ReadHost                    = (*Service)(nil)
	_ campaign.IntegrationHost             = (*Service)(nil)
	_ campaign.PatchHost                   = (*Service)(nil)
	_ campaign.PublishHost                 = (*Service)(nil)
	_ campaign.CloneHost                   = (*Service)(nil)
	_ campaign.MutationNotifyHost          = (*Service)(nil)
	_ campaign.MigrationHost               = (*Service)(nil)
	_ campaignworker.LoopHost              = (*Service)(nil)
	_ campaignworker.VPPHost               = (*Service)(nil)
	_ campaignworker.DrainHost             = (*Service)(nil)
	_ wizard.WizardHost                    = (*Service)(nil)
	_ campaign.TemplateCatalogHost         = (*Service)(nil)
	_ automation.LicenseGate               = (*Service)(nil)
)
