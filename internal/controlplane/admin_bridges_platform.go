package controlplane

import (
	"ad-event-processor/internal/billingadmin"
	"ad-event-processor/internal/campaign"
	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/edge"
	"ad-event-processor/internal/governance"
	"ad-event-processor/internal/licensing"
	"ad-event-processor/internal/licensingadmin"
	"ad-event-processor/internal/opsadmin"
	"ad-event-processor/internal/payment"
	"ad-event-processor/internal/platformadmin"
	"ad-event-processor/internal/privacyadmin"
	"ad-event-processor/internal/reconciliation"
	"ad-event-processor/internal/settingsadmin"
	"ad-event-processor/internal/shardadmin"
	"ad-event-processor/pkg/domainhealth"
	"ad-event-processor/pkg/legal"
	"ad-event-processor/pkg/platformconfig"
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type (
	auditCampaignFraudChange    = platformadmin.AuditCampaignFraudChange
	auditEmergencyBreakerChange = platformadmin.AuditEmergencyBreakerChange
	auditLicenseApplyChange     = platformadmin.AuditLicenseApplyChange
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

func (s *Service) StartVendorTelemetryWorker() {
	platformadmin.StartVendorTelemetryWorker(s)
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
	switch NormalizeRole(role) {
	case RoleTeamLead, RoleMediaBuyer:
		return NormalizeRole(role), nil
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

func (s *Service) InviteTeamMember(ctx context.Context, customerID uuid.UUID, email, role string) (TeamMemberDTO, error) {
	return s.teamGovernance().InviteTeamMember(ctx, customerID, email, role)
}

func (s *Service) UpdateTeamMember(ctx context.Context, customerID, userID uuid.UUID, in UpdateTeamMemberRequest) (TeamMemberDTO, error) {
	return s.teamGovernance().UpdateTeamMember(ctx, customerID, userID, in)
}

func (s *Service) ListTeamBudgetApprovals(ctx context.Context, customerID uuid.UUID) ([]TeamBudgetApprovalDTO, error) {
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

var (
	ErrBudgetApprovalRequired   = governance.ErrBudgetApprovalRequired
	ErrBudgetApprovalAutoDenied = governance.ErrBudgetApprovalAutoDenied
)

var NewQuotaManager = governance.NewQuotaManager

var (
	_ governance.Host     = (*Service)(nil)
	_ reconciliation.Host = (*Service)(nil)
)

func (s *Service) Config() *config.Config {
	if s == nil {
		return nil
	}
	return s.cfg
}

func (s *Service) SetSettlePool(pool *pgxpool.Pool) {
	if s == nil {
		return
	}
	s.settlementPostgresPool = pool
}

func (s *Service) SettlementPool() *pgxpool.Pool {
	if s == nil {
		return nil
	}
	if s.settlementPostgresPool != nil {
		return s.settlementPostgresPool
	}
	return s.GetPool()
}

func (s *Service) PaymentQueryPool() reconciliation.PaymentQueryer {
	if s == nil {
		return nil
	}
	if s.paymentPool != nil {
		return s.paymentPool
	}
	return s.GetPool()
}

func (s *Service) Sharder() domain.Sharder {
	if s == nil {
		return nil
	}
	return s.sharder
}

func (s *Service) Alerter() reconciliation.Alerter {
	if s == nil {
		return nil
	}
	return s.alerter
}

func (s *Service) BrokerDeltas() reconciliation.BrokerPendingDeltaReader {
	if s == nil {
		return nil
	}
	return s.brokerDeltas
}

func (s *Service) InvalidServiceFilterErr() error {
	return ErrInvalidServiceFilter
}

func (s *Service) checkMediaBuyerBudgetCap(ctx context.Context, userID, campaignID uuid.UUID, newLimit int64) error {
	return governance.CheckMediaBuyerBudgetCap(ctx, s.GetPool(), userID, campaignID, newLimit)
}

func (s *Service) CheckMediaBuyerBudgetCap(ctx context.Context, userID, campaignID uuid.UUID, newLimit int64) error {
	return s.checkMediaBuyerBudgetCap(ctx, userID, campaignID, newLimit)
}

func (s *Service) ListReconRuns(ctx context.Context, service string, limit, offset int32) ([]ReconRunDTO, int64, error) {
	return reconciliation.ListRuns(ctx, s, service, limit, offset)
}

func (s *Service) applyRegionSpendSyncBatch(ctx context.Context, batchDedupKey string, payload []byte) error {
	reconciler := s.GlobalSpendReconciler()
	if reconciler == nil {
		return nil
	}
	return reconciler.ApplyRegionSpendSyncBatch(ctx, batchDedupKey, payload)
}

func (s *Service) ForceRefillCampaignFromPG(ctx context.Context, campaignID uuid.UUID, currentSpend int64) error {
	if s == nil || s.GetPool() == nil {
		return errors.New("service unavailable")
	}
	var budgetLimit int64
	err := s.GetPool().QueryRow(ctx, `SELECT budget_limit FROM campaigns WHERE id = $1`, domain.ToUUID(campaignID)).Scan(&budgetLimit)
	if err != nil {
		return err
	}
	remaining := budgetLimit - currentSpend
	if remaining < 0 {
		remaining = 0
	}
	redisClient := s.RedisClientForCampaign(campaignID)
	if redisClient == nil {
		return fmt.Errorf("no redis shard for campaign %s", campaignID)
	}
	return redisClient.Set(ctx, domain.BudgetCampaignKey(campaignID), remaining, 0).Err()
}

func (s *Service) RedisClientForCampaign(campaignID uuid.UUID) redis.UniversalClient {
	return s.redisClientForCampaign(campaignID)
}

func (s *Service) RunStuckDrainCheck(ctx context.Context) {
	if s == nil {
		return
	}
	s.CheckStuckDrainJobs(ctx)
}

type licensingHost struct {
	svc *Service
}

func (s *Service) LicensingService() *licensingadmin.Service {
	if s == nil {
		return nil
	}
	return licensingadmin.NewService(licensingHost{svc: s})
}

func (h licensingHost) Pool() *pgxpool.Pool { return h.svc.GetPool() }

func (h licensingHost) ReloadLicense(ctx context.Context) error { return reloadLicense(ctx) }

func (h licensingHost) ActiveActivationLicenseKey() string { return activeActivationLicenseKey() }

func (h licensingHost) WorkerOpContext(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return workerContext(parent, timeout)
}

func (h licensingHost) ErrValidation(msg string) error { return errValidation(msg) }

func (h licensingHost) ActorUserID(ctx context.Context) uuid.UUID {
	if u, ok := GetUser(ctx); ok {
		return u.UserID
	}
	return uuid.Nil
}

func (h licensingHost) AuditLicenseApply(ctx context.Context, claims *licensing.LicenseClaims) {
	if h.svc == nil || claims == nil {
		return
	}
	adminID := h.ActorUserID(ctx)
	depID, err := uuid.Parse(claims.DeploymentID)
	var targetID *uuid.UUID
	if err == nil {
		targetID = &depID
	}
	change := platformadmin.AuditLicenseApplyChange{
		DeploymentID: claims.DeploymentID,
		ValidUntil:   claims.ValidUntil.UTC().Format(time.RFC3339),
		CustomerName: claims.CustomerName,
		Plan:         claims.Plan,
		Revoked:      claims.Revoked,
	}
	h.svc.AuditLog(ctx, nil, adminID, "LICENSE_APPLY", "license", targetID, change, nil)
	if h.svc.alerter != nil {
		h.svc.alerter.AlertLicenseApplied(ctx, claims.DeploymentID, claims.ValidUntil, adminID.String(), claims.Revoked)
	}
}

func (h licensingHost) DeploymentLimits() (licensing.Limits, licensing.LicenseState, bool) {
	return licenseDeploymentLimits()
}

func (h licensingHost) FeatureAllowed(featureKey string) (bool, string) {
	return licenseFeatureAllowed(featureKey)
}

func (h licensingHost) EulaPool() *pgxpool.Pool { return h.svc.GetPool() }

func (h licensingHost) EulaActorID(ctx context.Context) uuid.UUID { return h.ActorUserID(ctx) }

func (h licensingHost) EulaAuditAccept(ctx context.Context, q db.Querier, adminID uuid.UUID, version, acceptedBy string) {
	h.svc.AuditLog(ctx, q, adminID, "EULA_ACCEPT", "system", nil, map[string]string{
		"version": version,
		"by":      acceptedBy,
	}, nil)
}

func (s *Service) ApplyLicenseToken(ctx context.Context, token string) error {
	return s.LicensingService().ApplyLicenseToken(ctx, token)
}

func (s *Service) enforceDeploymentLicenseCampaignCap(ctx context.Context) error {
	return s.LicensingService().EnforceDeploymentCampaignCap(ctx)
}

func (s *Service) StartLicenseRevokeQueueWorker(interval time.Duration) {
	if s == nil || s.pool == nil {
		return
	}
	s.startWorker(func() {
		licensingadmin.NewRevokeQueueWorker(
			s.pool,
			interval,
			reloadLicense,
			activeActivationLicenseKey,
		).Start(s.ctx)
	})
}

func (s *Service) GetEulaStatus(ctx context.Context) (legal.Acceptance, bool, error) {
	return licensingadmin.GetEulaStatus(ctx, licensingHost{svc: s})
}

func (s *Service) AcceptEula(ctx context.Context, version, acceptedBy string) error {
	return licensingadmin.AcceptEula(ctx, licensingHost{svc: s}, version, acceptedBy)
}

func (s *Service) saveEulaAcceptanceTx(ctx context.Context, q db.Querier, version, acceptedBy string) error {
	return licensingadmin.SaveEulaAcceptanceTx(ctx, licensingHost{svc: s}, q, version, acceptedBy)
}

var _ licensingadmin.EulaHost = licensingHost{}

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
	return shardadmin.SyncGlobalSetReplaceToAllShards(ctx, s.redisShards, key, ifaceMembers)
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

var _ privacyadmin.Host = (*Service)(nil)

func (s *Service) PrivacyStore() *privacyadmin.Store {
	if s == nil {
		return nil
	}
	if s.privacyStore == nil {
		s.privacyStore = privacyadmin.NewStore(s.pool, s)
	}
	return s.privacyStore
}

func (s *Service) ConsentRetentionMonths() int {
	if s.cfg == nil {
		return 0
	}
	return s.cfg.ConsentRetentionMonths
}

func (s *Service) WorkerBatchContext(parent context.Context) (context.Context, context.CancelFunc) {
	return workerContext(parent, workerBatchTimeout)
}

func (s *Service) ClickHouseDeleteFraudEventsByUser(ctx context.Context, userID string) error {
	if s.clickhouseWriteConn == nil || userID == "" {
		return nil
	}
	query := `ALTER TABLE fraud_events DELETE WHERE user_id = ?`
	return s.clickhouseWriteConn.Exec(ctx, query, userID)
}

func (s *Service) PublishConsentUpdate(ctx context.Context, hashHex string) error {
	return shardadmin.PublishControlChannelToAllShards(ctx, s.redisShards, s.ConsentUpdateChannel(), hashHex)
}

func (s *Service) PurgeUserRedisKeys(ctx context.Context, hashHex, subjectUserID string) error {
	if len(s.redisShards) == 0 {
		return fmt.Errorf("no redis clients")
	}
	consentKey := domain.ConsentRedisKeyPrefix + hashHex
	pattern := "*:u:" + subjectUserID
	var firstErr error
	var success int
	for _, redisClient := range s.redisShards {
		if redisClient == nil {
			continue
		}
		if err := redisClient.Del(ctx, consentKey).Err(); err != nil && !errors.Is(err, redis.Nil) {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		success++
		iter := redisClient.Scan(ctx, 0, pattern, 200).Iterator()
		for iter.Next(ctx) {
			_ = redisClient.Del(ctx, iter.Val()).Err()
		}
		_ = iter.Err()
	}
	if success == 0 && firstErr != nil {
		return firstErr
	}
	return nil
}

func (s *Service) SyncConsentRedisKey(ctx context.Context, hashHex string, purposes int16) error {
	if len(s.redisShards) == 0 {
		return fmt.Errorf("no redis clients")
	}
	val := strconv.FormatInt(int64(purposes), 10)
	key := domain.ConsentRedisKeyPrefix + hashHex
	wrote := 0
	for _, redisClient := range s.redisShards {
		if redisClient == nil {
			continue
		}
		if err := redisClient.Set(ctx, key, val, 0).Err(); err != nil {
			return err
		}
		wrote++
	}
	if wrote == 0 {
		return fmt.Errorf("no connected redis shard for consent write")
	}
	return shardadmin.PublishControlChannelToAllShards(ctx, s.redisShards, s.ConsentUpdateChannel(), hashHex)
}

func (s *Service) ConsentUpdateChannel() string {
	return s.consentUpdateChannel()
}

func (s *Service) consentUpdateChannel() string {
	if s.cfg != nil && s.cfg.ConsentUpdateChannel != "" {
		return s.cfg.ConsentUpdateChannel
	}
	return domain.ConsentDefaultUpdateChannel
}

func VerifyConsentHMAC(secret []byte, body []byte, signatureHex string) error {
	return privacyadmin.VerifyConsentHMAC(secret, body, signatureHex)
}

func (s *Service) RecordConsent(ctx context.Context, in ConsentRecord) error {
	return s.PrivacyStore().RecordConsent(ctx, privacyadmin.ConsentRecord{
		UserID:   in.UserID,
		Source:   in.Source,
		Purposes: in.Purposes,
	})
}

func (s *Service) UpdateCampaignConsentRequirements(ctx context.Context, campaignID uuid.UUID, purposes int16) error {
	return s.PrivacyStore().UpdateCampaignConsentRequirements(ctx, campaignID, purposes)
}

func (s *Service) CleanupConsentEvents(ctx context.Context) error {
	return s.PrivacyStore().CleanupConsentEvents(ctx)
}

func (s *Service) CreatePrivacyErasureRequest(ctx context.Context, userID string) (uuid.UUID, error) {
	return s.PrivacyStore().CreatePrivacyErasureRequest(ctx, userID)
}

func (s *Service) ProcessPrivacyErasureTick(ctx context.Context) error {
	return s.PrivacyStore().ProcessPrivacyErasureTick(ctx)
}

func (s *Service) PurgeUserDataRedis(ctx context.Context, hashHex, subjectUserID string) error {
	return s.PrivacyStore().PurgeUserDataRedis(ctx, hashHex, subjectUserID)
}

func (s *Service) SyncUserConsentToRedis(ctx context.Context, hashHex string, purposes int16) error {
	return s.PrivacyStore().SyncUserConsentToRedis(ctx, hashHex, purposes)
}

func (s *Service) MarkErasureRedisPurgeDone(ctx context.Context, erasureID uuid.UUID, partialErr error) error {
	return s.PrivacyStore().MarkErasureRedisPurgeDone(ctx, erasureID, partialErr)
}

var (
	ErrConsentInvalidSignature = privacyadmin.ErrConsentInvalidSignature
	ErrConsentInvalidPayload   = privacyadmin.ErrConsentInvalidPayload
)

var _ opsadmin.ConsentRecorder = (*Service)(nil)

var (
	_ billingadmin.CryptoBillingHost = (*Service)(nil)
	_ billingadmin.UsageExportHost   = (*Service)(nil)
	_ billingadmin.TenantCapHost     = (*Service)(nil)
	_ billingadmin.CreditHost        = (*Service)(nil)
)

func (s *Service) PaymentPool() *pgxpool.Pool {
	if s == nil {
		return nil
	}
	return s.paymentPool
}

func (s *Service) FallbackPool() *pgxpool.Pool {
	if s == nil {
		return nil
	}
	return s.GetPool()
}

func (s *Service) ExportChunkMaxBytes() int {
	limits, state, ok := licenseDeploymentLimits()
	return billingadmin.ExportChunkMaxBytes(limits, state, ok)
}

func (s *Service) DeploymentLimits() (licensing.Limits, licensing.LicenseState, bool) {
	return licenseDeploymentLimits()
}

func (s *Service) CreditScoringConfig() billingadmin.CreditScoringConfig {
	if s == nil || s.cfg == nil {
		return billingadmin.CreditScoringConfig{}
	}
	return billingadmin.CreditScoringConfig{
		MinAgeDays:         s.cfg.CreditScoringMinAgeDays,
		MatureAgeDays:      s.cfg.CreditScoringMatureAgeDays,
		MidTierPercent:     s.cfg.CreditScoringMidTierPercent,
		MaturePercent:      s.cfg.CreditScoringMaturePercent,
		MaxCap:             s.cfg.CreditScoringMaxCap,
		ReconLagThreshold:  s.cfg.CreditScoringReconLagThreshold,
		ReconLagPenaltyPct: s.cfg.CreditScoringReconLagPenaltyPct,
	}
}

func (s *Service) enforceDeploymentTenantCap(ctx context.Context) error {
	return billingadmin.EnforceDeploymentTenantCap(ctx, s)
}

func (s *Service) ExportUsageDailyCSV(ctx context.Context, spec UsageExportSpec, w io.Writer) (UsageExportResult, error) {
	return billingadmin.ExportUsageDailyCSV(ctx, s, spec, w)
}

func (s *Service) ProcessCryptoWebhook(ctx context.Context, eventID, eventType string, payload []byte, providerRef string, amountMicro int64, txHash string, confirmations int) error {
	return billingadmin.ProcessCryptoWebhook(ctx, s, eventID, eventType, payload, providerRef, amountMicro, txHash, confirmations)
}

var (
	_ platformadmin.AuditWriterHost = (*Service)(nil)
	_ payment.SettlementHost        = (*Service)(nil)
	_ payment.CTVSettlementHost    = (*Service)(nil)
)

type Days = platformadmin.Days

type SettlementHandler = payment.SettlementHandler

func NewSettlementHandler(svc *Service, cfg *config.Config) *payment.SettlementHandler {
	return payment.NewSettlementHandler(svc, cfg)
}

func (s *Service) ApplyCTVSettlement(
	ctx context.Context,
	settlementID string,
	customerID, campaignID uuid.UUID,
	spendMicro int64,
) (domain.CTVSettlementResult, error) {
	return payment.ApplyCTVSettlement(ctx, s, settlementID, customerID, campaignID, spendMicro)
}

func (s *Service) ErrPaymentTopupNotFound() error { return ErrPaymentTopupNotFound }

func (s *Service) AuditLog(ctx context.Context, q db.Querier, adminID uuid.UUID, action, targetType string, targetID *uuid.UUID, changes, metadata any) {
	platformadmin.AuditLog(ctx, s, q, adminID, action, targetType, targetID, changes, metadata)
}

func (s *Service) RunAuditCleaner(ctx context.Context, retention Days) {
	platformadmin.RunAuditCleaner(ctx, s, retention)
}

func (s *Service) ListCustomers(ctx context.Context, limit, offset int32, sortField, sortOrder string) ([]CustomerDTO, int64, error) {
	return s.customersAdmin().ListCustomers(ctx, limit, offset, sortField, sortOrder)
}

func (s *Service) GetCustomerDTO(ctx context.Context, id uuid.UUID) (CustomerDTO, error) {
	return s.customersAdmin().GetCustomerDTO(ctx, id)
}

func (s *Service) UpdateCustomerCostCenter(ctx context.Context, customerID uuid.UUID, costCenter string) (CustomerDTO, error) {
	return s.customersAdmin().UpdateCustomerCostCenter(ctx, customerID, costCenter)
}

func (s *Service) ListCustomerLedger(ctx context.Context, customerID uuid.UUID, limit, offset int32) ([]LedgerDTO, int64, error) {
	return s.customerLedger().ListCustomerLedger(ctx, customerID, limit, offset)
}

func (s *Service) GetCustomerBalance(ctx context.Context, customerID uuid.UUID) (CustomerBalanceDTO, error) {
	return s.customerLedger().GetCustomerBalance(ctx, customerID)
}

func (s *Service) ExportCustomerLedgerCSV(ctx context.Context, customerID uuid.UUID, cursor int64, w io.Writer) (LedgerExportResult, error) {
	return s.customerLedger().ExportCustomerLedgerCSV(ctx, customerID, cursor, w)
}

func (s *Service) ListDisputes(ctx context.Context, customerFilter string, limit, offset int32) (DisputeListResult, error) {
	return s.disputesAdmin().ListDisputes(ctx, customerFilter, limit, offset)
}

func (s *Service) ListAuditLogs(ctx context.Context, limit, offset int32, redactPII bool) ([]AuditLogDTO, int64, error) {
	return platformadmin.ListAuditLogs(ctx, s, limit, offset, redactPII)
}

func (s *Service) RetryNotification(ctx context.Context, notificationID string) error {
	return platformadmin.RetryNotification(ctx, s, notificationID)
}

func (s *Service) WarmCampaignBudget(ctx context.Context, campaignID uuid.UUID) (int64, error) {
	return platformadmin.WarmCampaignBudget(ctx, s, campaignID)
}

func (s *Service) EnforceSelfServeCreateLimits(ctx context.Context, customerID uuid.UUID, budgetMicro int64) error {
	return platformadmin.EnforceSelfServeCreateLimits(ctx, s, customerID, budgetMicro)
}

func (s *Service) SupportFeedbackMeta(ctx context.Context) (SupportFeedbackMeta, error) {
	return platformadmin.GetSupportFeedbackMeta(ctx, s)
}

func (s *Service) RecordSupportFeedback(ctx context.Context, in SupportFeedbackRecord) (uuid.UUID, error) {
	return platformadmin.RecordSupportFeedback(ctx, s, in)
}

func (s *Service) StartProductTelemetryPulse() {
	platformadmin.StartProductTelemetryPulse(s)
}

func (s *Service) BuildStackHealthSnapshot(ctx context.Context) (opsadmin.StackHealthSnapshot, error) {
	return opsadmin.BuildStackHealthSnapshot(ctx, opsadmin.StackHealthDeps{
		Pool:          s.GetPool(),
		LicenseState:  licenseWatcherState,
		ClickHouseLag: s.clickHouseIngestionLag,
		ShardHealth:   s.GetShardHealth,
	})
}

func (s *Service) ListDomainHealth(ctx context.Context) ([]DomainHealthDTO, error) {
	return s.domainHealthAdmin().ListDomainHealth(ctx)
}

func (s *Service) AddCustomDomain(ctx context.Context, hostname string) (DomainHealthDTO, error) {
	return s.domainHealthAdmin().AddCustomDomain(ctx, hostname)
}

func (s *Service) DeleteCustomDomain(ctx context.Context, hostname string) error {
	return s.domainHealthAdmin().DeleteCustomDomain(ctx, hostname)
}

func (s *Service) ProbeDomainNow(ctx context.Context, hostname string) (DomainHealthDTO, error) {
	return s.domainHealthAdmin().ProbeDomainNow(ctx, hostname)
}

func (s *Service) SetupDomainSSL(ctx context.Context, hostname string) (DomainSSLSetupResult, error) {
	return s.domainHealthAdmin().SetupDomainSSL(ctx, hostname)
}

func (s *Service) IsTLSAllowed(ctx context.Context, hostname string) (bool, error) {
	return s.domainHealthAdmin().IsTLSAllowed(ctx, hostname)
}

func (s *Service) ParkDomain(ctx context.Context, req ParkDomainRequest) (ParkDomainResponse, error) {
	return s.domainHealthAdmin().ParkDomain(ctx, req)
}

func (s *Service) StartDomainHealthWorker(ctx context.Context, interval time.Duration) {
	platformadmin.StartDomainHealthWorker(ctx, s, interval)
}

func (s *Service) SetReputationChecker(c *domainhealth.ReputationChecker) {
	if s == nil {
		return
	}
	s.reputation = c
}

func (s *Service) ReputationChecker() *domainhealth.ReputationChecker {
	if s == nil {
		return nil
	}
	if s.reputation != nil {
		return s.reputation
	}
	if s.cfg == nil || !s.cfg.Management.DomainReputationEnabled {
		return nil
	}
	s.reputation = domainhealth.NewReputationChecker(domainhealth.ReputationConfig{
		SafeBrowsingAPIKey: string(s.cfg.Management.SafeBrowsingAPIKey),
		FacebookToken:      string(s.cfg.Management.FacebookGraphAccessToken),
		FacebookGraphBase:  s.cfg.Management.FacebookGraphAPIBase,
	})
	return s.reputation
}

func (s *Service) SetCloudflareAPI(api platformadmin.CloudflareAPI) {
	if s == nil {
		return
	}
	s.cloudflare = api
}

func (s *Service) CloudflareClient() platformadmin.DomainCloudflareClient {
	if s == nil {
		return nil
	}
	if s.cloudflare != nil {
		return s.cloudflare
	}
	if s.cfg == nil {
		return nil
	}
	return platformadmin.NewCloudflareClient(string(s.cfg.Management.CloudflareAPIToken), s.cfg.Management.CloudflareAPIBase)
}
