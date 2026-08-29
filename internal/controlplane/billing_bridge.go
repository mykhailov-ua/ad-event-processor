package controlplane

import (
	"context"
	"io"
	"time"

	"ad-event-processor/internal/billingadmin"
	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/licensing"
	"ad-event-processor/internal/opsadmin"
	"ad-event-processor/internal/payment"
	"ad-event-processor/internal/platformadmin"
	"ad-event-processor/pkg/domainhealth"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

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

func (s *Service) ExportUsageDailyCSV(ctx context.Context, spec billingadmin.UsageExportSpec, w io.Writer) (billingadmin.UsageExportResult, error) {
	return billingadmin.ExportUsageDailyCSV(ctx, s, spec, w)
}

func (s *Service) ProcessCryptoWebhook(ctx context.Context, eventID, eventType string, payload []byte, providerRef string, amountMicro int64, txHash string, confirmations int) error {
	return billingadmin.ProcessCryptoWebhook(ctx, s, eventID, eventType, payload, providerRef, amountMicro, txHash, confirmations)
}

var (
	_ platformadmin.AuditWriterHost = (*Service)(nil)
	_ payment.SettlementHost        = (*Service)(nil)
	_ payment.CTVSettlementHost     = (*Service)(nil)
)

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

func (s *Service) ListCustomers(ctx context.Context, limit, offset int32, sortField, sortOrder string) ([]platformadmin.CustomerDTO, int64, error) {
	return s.customersAdmin().ListCustomers(ctx, limit, offset, sortField, sortOrder)
}

func (s *Service) GetCustomerDTO(ctx context.Context, id uuid.UUID) (platformadmin.CustomerDTO, error) {
	return s.customersAdmin().GetCustomerDTO(ctx, id)
}

func (s *Service) UpdateCustomerCostCenter(ctx context.Context, customerID uuid.UUID, costCenter string) (platformadmin.CustomerDTO, error) {
	return s.customersAdmin().UpdateCustomerCostCenter(ctx, customerID, costCenter)
}

func (s *Service) ListCustomerLedger(ctx context.Context, customerID uuid.UUID, limit, offset int32) ([]platformadmin.BalanceLedgerDTO, int64, error) {
	return s.customerLedger().ListCustomerLedger(ctx, customerID, limit, offset)
}

func (s *Service) GetCustomerBalance(ctx context.Context, customerID uuid.UUID) (platformadmin.CustomerBalanceDTO, error) {
	return s.customerLedger().GetCustomerBalance(ctx, customerID)
}

func (s *Service) ExportCustomerLedgerCSV(ctx context.Context, customerID uuid.UUID, cursor int64, w io.Writer) (platformadmin.LedgerExportResult, error) {
	return s.customerLedger().ExportCustomerLedgerCSV(ctx, customerID, cursor, w)
}

func (s *Service) ListDisputes(ctx context.Context, customerFilter string, limit, offset int32) (billingadmin.DisputeListResult, error) {
	return s.disputesAdmin().ListDisputes(ctx, customerFilter, limit, offset)
}

func (s *Service) ListAuditLogs(ctx context.Context, limit, offset int32, redactPII bool) ([]platformadmin.AuditLogDTO, int64, error) {
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

func (s *Service) SupportFeedbackMeta(ctx context.Context) (platformadmin.SupportFeedbackMeta, error) {
	return platformadmin.GetSupportFeedbackMeta(ctx, s)
}

func (s *Service) RecordSupportFeedback(ctx context.Context, in platformadmin.SupportFeedbackRecord) (uuid.UUID, error) {
	return platformadmin.RecordSupportFeedback(ctx, s, in)
}

func (s *Service) StartProductTelemetryPulse(ctx context.Context) {
	platformadmin.StartProductTelemetryPulse(ctx, s)
}

func (s *Service) BuildStackHealthSnapshot(ctx context.Context) (opsadmin.StackHealthSnapshot, error) {
	return opsadmin.BuildStackHealthSnapshot(ctx, opsadmin.StackHealthDeps{
		Pool:          s.GetPool(),
		LicenseState:  licenseWatcherState,
		ClickHouseLag: s.clickHouseIngestionLag,
		ShardHealth:   s.GetShardHealth,
	})
}

func (s *Service) ListDomainHealth(ctx context.Context) ([]platformadmin.DomainHealthDTO, error) {
	return s.domainHealthAdmin().ListDomainHealth(ctx)
}

func (s *Service) AddCustomDomain(ctx context.Context, hostname string) (platformadmin.DomainHealthDTO, error) {
	return s.domainHealthAdmin().AddCustomDomain(ctx, hostname)
}

func (s *Service) DeleteCustomDomain(ctx context.Context, hostname string) error {
	return s.domainHealthAdmin().DeleteCustomDomain(ctx, hostname)
}

func (s *Service) ProbeDomainNow(ctx context.Context, hostname string) (platformadmin.DomainHealthDTO, error) {
	return s.domainHealthAdmin().ProbeDomainNow(ctx, hostname)
}

func (s *Service) SetupDomainSSL(ctx context.Context, hostname string) (platformadmin.DomainSSLSetupResult, error) {
	return s.domainHealthAdmin().SetupDomainSSL(ctx, hostname)
}

func (s *Service) IsTLSAllowed(ctx context.Context, hostname string) (bool, error) {
	return s.domainHealthAdmin().IsTLSAllowed(ctx, hostname)
}

func (s *Service) ParkDomain(ctx context.Context, req platformadmin.ParkDomainRequest) (platformadmin.ParkDomainResponse, error) {
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
