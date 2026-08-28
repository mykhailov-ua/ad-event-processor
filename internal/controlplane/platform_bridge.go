package controlplane

import (
	"context"
	"crypto/subtle"

	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/platformadmin"
	"ad-event-processor/pkg/platformconfig"

	"github.com/google/uuid"
)

var _ platformadmin.Host = (*Service)(nil)

func (s *Service) PlatformStore() *platformadmin.Store {
	if s == nil {
		return nil
	}
	if s.platformStore == nil {
		s.platformStore = platformadmin.NewStore(s.pool, s)
	}
	return s.platformStore
}

func (s *Service) VerifyInstallToken(token string) error {
	if s == nil || s.cfg == nil || len(s.cfg.InstallBootstrapToken) == 0 {
		return platformadmin.ErrInstallTokenInvalid
	}
	if subtle.ConstantTimeCompare([]byte(token), []byte(s.cfg.InstallBootstrapToken)) != 1 {
		return platformadmin.ErrInstallTokenInvalid
	}
	return nil
}

func (s *Service) SaveBootstrapEula(ctx context.Context, q db.Querier, version, acceptedBy string) error {
	return s.saveEulaAcceptanceTx(ctx, q, version, acceptedBy)
}

func (s *Service) AuditBootstrap(ctx context.Context, q db.Querier, adminID uuid.UUID, cfg platformconfig.Config) {
	s.AuditLog(ctx, q, adminID, "PLATFORM_CONFIG_BOOTSTRAP", "system", nil, platformconfig.RedactConfig(cfg), nil)
}

func (s *Service) AuditUpdate(ctx context.Context, q db.Querier, adminID uuid.UUID, before, after platformconfig.Config, restartRequired []string) {
	s.AuditLog(ctx, q, adminID, "PLATFORM_CONFIG_UPDATE", "system", nil, map[string]any{
		"before":           platformconfig.RedactConfig(before),
		"after":            platformconfig.RedactConfig(after),
		"restart_required": restartRequired,
	}, nil)
}

func (s *Service) AuditApply(ctx context.Context, q db.Querier, adminID uuid.UUID, writtenPath string) {
	s.AuditLog(ctx, q, adminID, "PLATFORM_CONFIG_APPLY", "system", nil, map[string]string{
		"written_path": writtenPath,
	}, nil)
}

func (s *Service) SyncEdgeExpose(ctx context.Context, cfg platformconfig.Config) error {
	if s == nil || len(s.redisShards) == 0 {
		return nil
	}
	return syncGlobalConfigToAllShards(ctx, s.redisShards, platformconfig.EdgeExposeRedisSettings(cfg), 0)
}

func (s *Service) GetPlatformConfig(ctx context.Context) (platformconfig.Config, bool, error) {
	return s.PlatformStore().GetConfig(ctx)
}

func (s *Service) GetConfig(ctx context.Context) (platformconfig.Config, bool, error) {
	return s.PlatformStore().GetConfig(ctx)
}

func (s *Service) GetPlatformRestartPending(ctx context.Context) ([]string, error) {
	return s.PlatformStore().GetRestartPending(ctx)
}

func (s *Service) GetRestartPending(ctx context.Context) ([]string, error) {
	return s.PlatformStore().GetRestartPending(ctx)
}

func (s *Service) BootstrapPlatformConfig(ctx context.Context, req platformconfig.BootstrapRequest, installToken string) error {
	return s.PlatformStore().Bootstrap(ctx, req, installToken)
}

func (s *Service) Bootstrap(ctx context.Context, req platformconfig.BootstrapRequest, installToken string) error {
	return s.PlatformStore().Bootstrap(ctx, req, installToken)
}

func (s *Service) UpdatePlatformConfig(ctx context.Context, patch platformconfig.Patch) (platformconfig.Config, []string, error) {
	return s.PlatformStore().Update(ctx, patch)
}

func (s *Service) Update(ctx context.Context, patch platformconfig.Patch) (platformconfig.Config, []string, error) {
	return s.PlatformStore().Update(ctx, patch)
}

func (s *Service) ApplyPlatformConfig(ctx context.Context, installRoot string) (string, error) {
	return s.PlatformStore().Apply(ctx, installRoot)
}

func (s *Service) Apply(ctx context.Context, installRoot string) (string, error) {
	return s.PlatformStore().Apply(ctx, installRoot)
}
