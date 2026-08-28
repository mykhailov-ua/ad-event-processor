package controlplane

import (
	ctrlhttp "ad-event-processor/internal/control/http"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/platformadmin"
	"ad-event-processor/internal/shardadmin"
	"ad-event-processor/pkg/platformconfig"
	"context"
	"crypto/subtle"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type (
	auditEmergencyBreakerChange = platformadmin.AuditEmergencyBreakerChange
)

var (
	_ platformadmin.Host                   = (*Service)(nil)
	_ platformadmin.Service                = (*Service)(nil)
	_ platformadmin.MetaEnrichHost         = (*Service)(nil)
	_ platformadmin.CustomersHost          = (*Service)(nil)
	_ platformadmin.DomainHealthHost       = (*Service)(nil)
	_ platformadmin.CustomerLedgerHost     = (*Service)(nil)
	_ platformadmin.DisputesHost           = (*Service)(nil)
	_ platformadmin.AuditLogsHost          = (*Service)(nil)
	_ platformadmin.SupportFeedbackHost    = (*Service)(nil)
	_ platformadmin.SelfServeLimitsHost    = (*Service)(nil)
	_ platformadmin.CampaignBudgetWarmHost = (*Service)(nil)
	_ platformadmin.NotificationsHost      = (*Service)(nil)
	_ platformadmin.VendorTelemetryHost    = (*Service)(nil)
	_ platformadmin.ProductTelemetryHost   = (*Service)(nil)
)

func (s *Service) customersAdmin() *platformadmin.Customers {
	return platformadmin.NewCustomers(s)
}

func (s *Service) customerLedger() *platformadmin.CustomerLedger {
	return platformadmin.NewCustomerLedger(s)
}

func (s *Service) disputesAdmin() *platformadmin.Disputes {
	return platformadmin.NewDisputes(s)
}

func (s *Service) domainHealthAdmin() *platformadmin.DomainHealth {
	return platformadmin.NewDomainHealth(s)
}

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
	return shardadmin.SyncGlobalConfigToAllShards(ctx, s.redisShards, platformconfig.EdgeExposeRedisSettings(cfg), 0)
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

func (s *Service) CampaignRemainingBudget(ctx context.Context, campaignID uuid.UUID) (int64, error) {
	return NewOutboxWorker(s).CampaignRemainingBudget(ctx, campaignID)
}

func (s *Service) SetCampaignBudgetRemaining(ctx context.Context, pipe redis.Pipeliner, campaignIDStr string, campaignID uuid.UUID, payloadLimit int64) error {
	return NewOutboxWorker(s).SetCampaignBudgetRemaining(ctx, pipe, campaignIDStr, campaignID, payloadLimit)
}

func (s *Service) Apply(ctx context.Context, installRoot string) (string, error) {
	return s.PlatformStore().Apply(ctx, installRoot)
}

func (s *Service) ListPaymentDisputes(ctx context.Context, customerFilter string, limit, offset int32) (domain.ListDisputesResult, error) {
	if s == nil || s.payment == nil {
		return domain.ListDisputesResult{}, status.Error(codes.Unavailable, "payment service not configured")
	}
	return s.payment.ListDisputes(ctx, customerFilter, limit, offset)
}

func (s *Service) SelfServeBudgetMinMicro() int64 {
	if s == nil || s.cfg == nil {
		return 0
	}
	return s.cfg.SelfServeBudgetMinMicro
}

func (s *Service) SelfServeBudgetMaxMicro() int64 {
	if s == nil || s.cfg == nil {
		return 0
	}
	return s.cfg.SelfServeBudgetMaxMicro
}

func (s *Service) SelfServeMaxActiveCampaigns() int {
	if s == nil || s.cfg == nil {
		return 0
	}
	return s.cfg.SelfServeMaxActiveCampaigns
}

func (s *Service) SelfServeMaxCreatesPerDay() int {
	if s == nil || s.cfg == nil {
		return 0
	}
	return s.cfg.SelfServeMaxCreatesPerDay
}

func (s *Service) VendorTelemetryEnabled() bool {
	return s != nil && s.cfg != nil && s.cfg.VendorTelemetryEnabled
}

func (s *Service) VendorTelemetryInterval() time.Duration {
	if s == nil || s.cfg == nil {
		return 0
	}
	return time.Duration(s.cfg.VendorTelemetryIntervalSec) * time.Second
}

func (s *Service) VendorTelemetryTimeout() time.Duration {
	if s == nil || s.cfg == nil {
		return 0
	}
	return time.Duration(s.cfg.VendorTelemetryTimeoutSec) * time.Second
}

func (s *Service) GeoIPDBPath() string {
	if s == nil || s.cfg == nil {
		return ""
	}
	return s.cfg.GeoIP.DBPath
}

func (s *Service) StripeSecretKey() string {
	if s == nil || s.cfg == nil {
		return ""
	}
	return string(s.cfg.StripeSecretKey)
}

func (s *Service) TelegramBotToken() string {
	if s == nil || s.cfg == nil {
		return ""
	}
	return string(s.cfg.Notifier.TelegramBotToken)
}

func (s *Service) SMTPHost() string {
	if s == nil || s.cfg == nil {
		return ""
	}
	return s.cfg.Notifier.SMTPHost
}

func (s *Service) SMTPPort() string {
	if s == nil || s.cfg == nil {
		return ""
	}
	return s.cfg.Notifier.SMTPPort
}

func (s *Service) StartWorker(fn func()) {
	s.StartBackgroundWorker(fn)
}

func (s *Service) WorkerContext() context.Context {
	if s == nil || s.ctx == nil {
		return context.Background()
	}
	return s.ctx
}

func (s *Service) StartVendorTelemetryWorker(ctx context.Context) {
	platformadmin.StartVendorTelemetryWorker(ctx, s)
}

func (s *Service) TelemetryOptIn() bool {
	return s != nil && s.cfg != nil && s.cfg.TelemetryOptIn
}

func (s *Service) TelemetryURL() string {
	if s == nil || s.cfg == nil {
		return ""
	}
	return string(s.cfg.TelemetryURL)
}

func (s *Service) TelemetryInterval() time.Duration {
	if s == nil || s.cfg == nil {
		return 0
	}
	return time.Duration(s.cfg.TelemetryIntervalSec) * time.Second
}

func (s *Service) TelemetryHTTPTimeout() time.Duration {
	if s == nil || s.cfg == nil {
		return 0
	}
	return time.Duration(s.cfg.TelemetryHTTPTimeoutSec) * time.Second
}

var (
	_ platformadmin.GovernanceHost     = (*Service)(nil)
	_ platformadmin.BudgetApprovalHost = (*Service)(nil)
	_ platformadmin.TeamGovernance     = (*Service)(nil)
)

func (s *Service) teamGovernance() *platformadmin.Governance {
	return platformadmin.NewGovernance(s)
}

func (s *Service) NormalizeTeamRole(role string) (string, error) {
	switch ctrlhttp.NormalizeRole(role) {
	case ctrlhttp.RoleTeamLead, ctrlhttp.RoleMediaBuyer:
		return ctrlhttp.NormalizeRole(role), nil
	default:
		return "", errValidation("role must be TL or MB")
	}
}

func (s *Service) ErrCustomerNotFound() error { return ErrCustomerNotFound }

func (s *Service) ErrTeamMemberNotFound() error { return ErrTeamMemberNotFound }

func (s *Service) ErrCampaignNotFound() error { return ErrCampaignNotFound }

func (s *Service) ErrBudgetApprovalRequired() error { return ErrBudgetApprovalRequired }

func (s *Service) ErrBudgetApprovalAutoDenied() error { return ErrBudgetApprovalAutoDenied }

func (s *Service) MapCustomerNotFound(err error) error {
	return mapNotFound(err, ErrCustomerNotFound)
}

func (s *Service) MapCampaignNotFound(err error) error {
	return mapNotFound(err, ErrCampaignNotFound)
}

func (s *Service) InviteTeamMember(ctx context.Context, customerID uuid.UUID, email, role string) (platformadmin.TeamMemberDTO, error) {
	return s.teamGovernance().InviteTeamMember(ctx, customerID, email, role)
}

func (s *Service) UpdateTeamMember(ctx context.Context, customerID, userID uuid.UUID, in platformadmin.UpdateTeamMemberRequest) (platformadmin.TeamMemberDTO, error) {
	return s.teamGovernance().UpdateTeamMember(ctx, customerID, userID, in)
}

func (s *Service) ListTeamBudgetApprovals(ctx context.Context, customerID uuid.UUID) ([]platformadmin.TeamBudgetApprovalDTO, error) {
	return s.teamGovernance().ListTeamBudgetApprovals(ctx, customerID)
}

func (s *Service) ResolveTeamBudgetApproval(ctx context.Context, customerID, approvalID, resolverID uuid.UUID, approve bool) error {
	return s.teamGovernance().ResolveTeamBudgetApproval(ctx, customerID, approvalID, resolverID, approve)
}

func (s *Service) AssignCampaignOwner(ctx context.Context, campaignID, ownerUserID uuid.UUID) error {
	return platformadmin.AssignCampaignOwner(ctx, s, campaignID, ownerUserID)
}

func (s *Service) handleMediaBuyerBudgetIncrease(ctx context.Context, locked db.Campaign, userID uuid.UUID, newLimit int64) error {
	return platformadmin.HandleMediaBuyerBudgetIncrease(ctx, s, locked, userID, newLimit)
}
