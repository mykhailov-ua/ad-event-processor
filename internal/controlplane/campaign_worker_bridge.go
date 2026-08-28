package controlplane

import (
	"context"
	"time"

	"ad-event-processor/internal/campaign"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type deliveryOutboxMerge = campaign.DeliveryOutboxMerge

const (
	outboxPriSyncBrandCreatives = campaign.OutboxPriSyncBrandCreatives
	outboxPriCreateCampaign     = campaign.OutboxPriCreateCampaign
	outboxPriPacing             = campaign.OutboxPriPacing
	outboxPriPause              = campaign.OutboxPriPause
)

func (s *Service) CampaignWorker() *campaign.Worker {
	if s == nil {
		return nil
	}
	if s.campaignWorker == nil {
		s.campaignWorker = campaign.NewWorker(s.CampaignRuntime(), s)
	}
	return s.campaignWorker
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
	s.AuditLog(ctx, q, uuid.Nil, "PACING_LOOP_ADJUSTMENT", "campaign", &campaignID, auditPacingLoopAdjustment{
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

func (s *Service) ListCampaignTemplates(ctx context.Context, customerID uuid.UUID, limit, offset int32) ([]CampaignTemplateDTO, int64, error) {
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

func (s *Service) fetchPacingHourWeights(ctx context.Context) [24]float64 {
	if s.clickhouseQuery == nil {
		return campaign.UniformHourWeights()
	}
	lookbackEnd := time.Now().UTC().Truncate(time.Hour)
	lookbackStart := lookbackEnd.Add(-campaign.PacingLookbackDays * 24 * time.Hour)
	_, samples, err := s.queryForecastHourlySamples(ctx, lookbackStart, lookbackEnd, nil)
	if err != nil {
		return campaign.UniformHourWeights()
	}
	return buildHourWeights(samples)
}
