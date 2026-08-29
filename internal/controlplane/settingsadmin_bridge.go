package controlplane

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"ad-event-processor/internal/database"
	"ad-event-processor/internal/smartalerts"

	"github.com/jackc/pgx/v5/pgxpool"

	"ad-event-processor/internal/campaign"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/edge"
	"ad-event-processor/internal/opsadmin"
	"ad-event-processor/internal/settingsadmin"
	"ad-event-processor/internal/shardadmin"

	"github.com/google/uuid"
)

var (
	_ settingsadmin.Host      = (*Service)(nil)
	_ opsadmin.BlacklistAdmin = (*Service)(nil)
)

func (s *Service) SettingsStore() *settingsadmin.Store {
	if s == nil {
		return nil
	}
	if s.settingsStore == nil {
		s.settingsStore = settingsadmin.NewStore(s.pool, s)
	}
	return s.settingsStore
}

func (s *Service) IsProtectedIP(ip string) bool {
	return edge.IsProtected(ip)
}

func (s *Service) BlacklistAutoTTLHours() int {
	if s.cfg != nil && s.cfg.Management.BlacklistAutoTTLHours > 0 {
		return s.cfg.Management.BlacklistAutoTTLHours
	}
	return 24
}

func (s *Service) BlacklistFraudTTLHours() int {
	if s.cfg != nil && s.cfg.Management.BlacklistFraudTTLHours > 0 {
		return s.cfg.Management.BlacklistFraudTTLHours
	}
	return 168
}

func (s *Service) SyncGlobalSetReplace(ctx context.Context, key string, members []any) error {
	if len(s.redisShards) == 0 {
		return errors.New("no redis client available")
	}
	return shardadmin.SyncGlobalSetReplaceToAllShards(ctx, s.redisShards, key, members)
}

func (s *Service) SyncGlobalConfig(ctx context.Context, settings map[string]string) error {
	if len(s.redisShards) == 0 {
		return errors.New("no redis client available")
	}
	return shardadmin.SyncGlobalConfigToAllShards(ctx, s.redisShards, settings, 0)
}

func (s *Service) ReplicateConfigVersionFromPrimary(ctx context.Context) error {
	return shardadmin.ReplicateConfigVersionFromPrimary(ctx, s.redisShards)
}

func (s *Service) NewBlockIPPreview(change settingsadmin.BlockIPPreviewChange) (campaign.MutationPreviewDTO, error) {
	return campaign.NewMutationPreview("BLOCK_IP", change)
}

func (s *Service) AuditEmergencyBreaker(ctx context.Context, q db.Querier, adminID uuid.UUID, active bool, reason string) {
	s.AuditLog(ctx, q, adminID, "EMERGENCY_BREAKER_TOGGLED", "system", nil, auditEmergencyBreakerChange{
		Active: active,
		Reason: reason,
	}, nil)
}

func (s *Service) BlockIP(ctx context.Context, ip string, source string) error {
	return s.SettingsStore().BlockIP(ctx, ip, source)
}

func (s *Service) PreviewBlockIP(ctx context.Context, ip string, source string, ttlSeconds *int64) (campaign.MutationPreviewDTO, error) {
	return s.SettingsStore().PreviewBlockIP(ctx, ip, source, ttlSeconds)
}

func (s *Service) BlockIPWithTTL(ctx context.Context, ip string, source string, ttlSeconds *int64) error {
	return s.SettingsStore().BlockIPWithTTL(ctx, ip, source, ttlSeconds)
}

func (s *Service) EnqueueFraudThreat(ctx context.Context, action, ip, campaignID string, score float64, boost int32, ttlSeconds int64) error {
	return s.SettingsStore().EnqueueFraudThreat(ctx, action, ip, campaignID, score, boost, ttlSeconds)
}

func (s *Service) EnqueueFraudThreatBatch(ctx context.Context, items []opsadmin.FraudThreatEnqueueItem) (int, error) {
	batch := make([]settingsadmin.FraudThreatItem, len(items))
	for i, item := range items {
		batch[i] = settingsadmin.FraudThreatItem(item)
	}
	return s.SettingsStore().EnqueueFraudThreatBatch(ctx, batch)
}

func (s *Service) UnblockExpiredBlacklist(ctx context.Context, rows []db.ListExpiredBlacklistIPsRow) (int, error) {
	return s.SettingsStore().UnblockExpiredBlacklist(ctx, rows)
}

func (s *Service) UnblockIP(ctx context.Context, ip string, source string) error {
	return s.SettingsStore().UnblockIP(ctx, ip, source)
}

func (s *Service) UpdateSettings(ctx context.Context, settings map[string]string) error {
	return s.SettingsStore().UpdateSettings(ctx, settings)
}

func (s *Service) ListBlacklist(ctx context.Context, limit, offset int32) ([]campaign.BlacklistDTO, int64, error) {
	return s.SettingsStore().ListBlacklist(ctx, limit, offset)
}

func (s *Service) GetSettings(ctx context.Context) (map[string]string, error) {
	return s.SettingsStore().GetSettings(ctx)
}

func (s *Service) SyncSystemState(ctx context.Context) error {
	return s.SettingsStore().SyncSystemState(ctx)
}

func (s *Service) ToggleEmergencyBreaker(ctx context.Context, active bool, reason string) error {
	return s.SettingsStore().ToggleEmergencyBreaker(ctx, active, reason)
}

var _ opsadmin.BlacklistAdmin = (*Service)(nil)

type SmartAlertsHTTPHandlers = smartalerts.HTTPHandlers

func (s *Service) SmartAlertsStore() *smartalerts.Store {
	if s == nil {
		return nil
	}
	return smartalerts.NewStore(smartalertsHost{s})
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
