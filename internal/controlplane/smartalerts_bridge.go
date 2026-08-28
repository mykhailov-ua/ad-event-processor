package controlplane

import (
	"context"
	"log/slog"
	"time"

	"ad-event-processor/internal/database"
	"ad-event-processor/internal/smartalerts"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

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
