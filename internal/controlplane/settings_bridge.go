package controlplane

import (
	"context"
	"errors"

	"ad-event-processor/internal/campaign"
	"ad-event-processor/internal/edge"
	"ad-event-processor/internal/opsadmin"
	"ad-event-processor/internal/settingsadmin"

	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
)

var _ settingsadmin.Host = (*Service)(nil)

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
	ifaceMembers := make([]interface{}, len(members))
	for i, m := range members {
		ifaceMembers[i] = m
	}
	return syncGlobalSetReplaceToAllShards(ctx, s.redisShards, key, ifaceMembers)
}

func (s *Service) SyncGlobalConfig(ctx context.Context, settings map[string]string) error {
	if len(s.redisShards) == 0 {
		return errors.New("no redis client available")
	}
	return syncGlobalConfigToAllShards(ctx, s.redisShards, settings, 0)
}

func (s *Service) ReplicateConfigVersionFromPrimary(ctx context.Context) error {
	return replicateConfigVersionFromPrimary(ctx, s.redisShards)
}

func (s *Service) NewBlockIPPreview(change settingsadmin.BlockIPPreviewChange) (campaign.MutationPreviewDTO, error) {
	return newMutationPreview("BLOCK_IP", BlockIPWouldChange(change))
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

func (s *Service) PreviewBlockIP(ctx context.Context, ip string, source string, ttlSeconds *int64) (MutationPreview, error) {
	return s.SettingsStore().PreviewBlockIP(ctx, ip, source, ttlSeconds)
}

func (s *Service) BlockIPWithTTL(ctx context.Context, ip string, source string, ttlSeconds *int64) error {
	return s.SettingsStore().BlockIPWithTTL(ctx, ip, source, ttlSeconds)
}

func (s *Service) EnqueueFraudThreat(ctx context.Context, action, ip, campaignID string, score float64, boost int32, ttlSeconds int64) error {
	return s.SettingsStore().EnqueueFraudThreat(ctx, action, ip, campaignID, score, boost, ttlSeconds)
}

func (s *Service) EnqueueFraudThreatBatch(ctx context.Context, items []FraudThreatEnqueueItem) (int, error) {
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

func (s *Service) ListBlacklist(ctx context.Context, limit, offset int32) ([]BlacklistDTO, int64, error) {
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
