#!/usr/bin/env python3
import pathlib
import subprocess
import sys

ROOT = pathlib.Path(__file__).resolve().parents[2] / "internal" / "controlplane"

PLATFORM_BRIDGE = pathlib.Path(__file__).with_name("_platform_bridge_full.go.txt")
LICENSING_BRIDGE = pathlib.Path(__file__).with_name("_licensing_bridge.go.txt")
PRIVACY_BRIDGE = pathlib.Path(__file__).with_name("_privacy_bridge.go.txt")

PLATFORM_GOVERNANCE = """package controlplane

import (
	"context"

	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/platformadmin"

	"github.com/google/uuid"
)

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
"""

GOVERNANCE_BRIDGE = """package controlplane

import (
	"context"
	"errors"
	"fmt"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/governance"
	"ad-event-processor/internal/reconciliation"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

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
"""

BILLING_BRIDGE = """package controlplane

import (
	"context"
	"io"

	"ad-event-processor/internal/billingadmin"
	"ad-event-processor/internal/licensing"

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

func (s *Service) enforceDeploymentTenantCap(ctx context.Context) error {
	return billingadmin.EnforceDeploymentTenantCap(ctx, s)
}

func (s *Service) ExportUsageDailyCSV(ctx context.Context, spec UsageExportSpec, w io.Writer) (UsageExportResult, error) {
	return billingadmin.ExportUsageDailyCSV(ctx, s, spec, w)
}

func (s *Service) ProcessCryptoWebhook(ctx context.Context, eventID, eventType string, payload []byte, providerRef string, amountMicro int64, txHash string, confirmations int) error {
	return billingadmin.ProcessCryptoWebhook(ctx, s, eventID, eventType, payload, providerRef, amountMicro, txHash, confirmations)
}
"""

PAYMENT_BRIDGE = """package controlplane

import "ad-event-processor/internal/payment"

var _ payment.SettlementHost = (*Service)(nil)

func (s *Service) ErrPaymentTopupNotFound() error {
	return ErrPaymentTopupNotFound
}
"""

WORKSPACE_BILLING = """package controlplane

import (
	"context"

	"ad-event-processor/internal/billingadmin"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
)

func (s *Service) UpdateCustomerCostCenter(ctx context.Context, customerID uuid.UUID, costCenter string) (CustomerDTO, error) {
	normalized, err := billingadmin.NormalizeCostCenter(costCenter)
	if err != nil {
		return CustomerDTO{}, err
	}
	q := db.New(s.GetPool())
	if _, err := q.GetCustomerByID(ctx, domain.ToUUID(customerID)); err != nil {
		return CustomerDTO{}, mapNotFound(err, ErrCustomerNotFound)
	}
	if _, err := q.UpdateCustomerCostCenter(ctx, db.UpdateCustomerCostCenterParams{
		ID:         domain.ToUUID(customerID),
		CostCenter: normalized,
	}); err != nil {
		return CustomerDTO{}, err
	}
	return s.GetCustomerDTO(ctx, customerID)
}
"""

MERGE_FILES = [
    "platform_bridge.go",
    "platform_governance_bridge.go",
    "governance_bridge.go",
    "licensing_bridge.go",
    "settings_bridge.go",
    "privacy_bridge.go",
    "payment_bridge.go",
    "billing_bridge.go",
]

EXTRA_DELETE = [
    "telemetry_pulse_bridge.go",
    "vendor_telemetry_bridge.go",
    "vendor_telemetry.go",
]


def strip_pkg_imports(text: str) -> str:
    out: list[str] = []
    skip = False
    for line in text.splitlines():
        if line.startswith("package "):
            continue
        if line.startswith("import "):
            skip = True
            continue
        if skip:
            if line.strip() == ")":
                skip = False
            continue
        out.append(line)
    return "\n".join(out).strip() + "\n"


def patch_platform_bridge(text: str) -> str:
    if "customersAdmin" not in text:
        sys.exit("platform_bridge.go missing customersAdmin")
    if "CampaignRemainingBudget" not in text:
        insert = """
func (s *Service) CampaignRemainingBudget(ctx context.Context, campaignID uuid.UUID) (int64, error) {
	return NewOutboxWorker(s).CampaignRemainingBudget(ctx, campaignID)
}

func (s *Service) SetCampaignBudgetRemaining(ctx context.Context, pipe redis.Pipeliner, campaignIDStr string, campaignID uuid.UUID, payloadLimit int64) error {
	return NewOutboxWorker(s).SetCampaignBudgetRemaining(ctx, pipe, campaignIDStr, campaignID, payloadLimit)
}
"""
        marker = "func (s *Service) Apply(ctx context.Context, installRoot string)"
        if marker in text:
            text = text.replace(marker, insert + "\n" + marker, 1)
        if '"github.com/redis/go-redis/v9"' not in text:
            text = text.replace(
                '"github.com/google/uuid"\n)',
                '"github.com/google/uuid"\n\t"github.com/redis/go-redis/v9"\n)',
                1,
            )
    if "auditCampaignFraudChange" not in text:
        alias = """
type (
	auditCampaignFraudChange    = platformadmin.AuditCampaignFraudChange
	auditEmergencyBreakerChange = platformadmin.AuditEmergencyBreakerChange
	auditLicenseApplyChange     = platformadmin.AuditLicenseApplyChange
)

"""
        text = text.replace("var (\n\t_ platformadmin.Host", alias + "var (\n\t_ platformadmin.Host", 1)
    return text


def main() -> int:
    (ROOT / "platform_bridge.go").write_text(PLATFORM_BRIDGE.read_text())
    (ROOT / "licensing_bridge.go").write_text(LICENSING_BRIDGE.read_text())
    (ROOT / "privacy_bridge.go").write_text(PRIVACY_BRIDGE.read_text())
    platform_path = ROOT / "platform_bridge.go"
    if not platform_path.exists():
        print("missing platform_bridge.go", file=sys.stderr)
        return 1

    (ROOT / "platform_governance_bridge.go").write_text(PLATFORM_GOVERNANCE)
    (ROOT / "governance_bridge.go").write_text(GOVERNANCE_BRIDGE)
    (ROOT / "billing_bridge.go").write_text(BILLING_BRIDGE)
    (ROOT / "payment_bridge.go").write_text(PAYMENT_BRIDGE)
    (ROOT / "service_workspace_billing.go").write_text(WORKSPACE_BILLING)

    platform_text = patch_platform_bridge(platform_path.read_text())
    platform_path.write_text(platform_text)

    audit_types = ROOT / "audit_types.go"
    at = audit_types.read_text()
    at = at.replace(
        "type auditEmergencyBreakerChange = platformadmin.AuditEmergencyBreakerChange\n\n",
        "",
    )
    if "type auditCampaignFraudChange struct" in at:
        import re

        at = re.sub(
            r"type auditCampaignFraudChange struct \{[^}]+\}\n\n",
            "",
            at,
            count=1,
        )
    if "type auditLicenseApplyChange struct" in at:
        import re

        at = re.sub(
            r"type auditLicenseApplyChange struct \{[^}]+\}\n",
            "",
            at,
            count=1,
        )
    audit_types.write_text(at)

    out_lines = ["package controlplane", ""]
    for name in MERGE_FILES:
        path = ROOT / name
        if not path.exists():
            print(f"missing {name}", file=sys.stderr)
            return 1
        out_lines.append(strip_pkg_imports(path.read_text()).rstrip())
        out_lines.append("")

    out_path = ROOT / "admin_bridges_platform.go"
    merged = "\n".join(out_lines).rstrip() + "\n"
    merged = merged.replace(
        "var _ opsadmin.BlacklistAdmin = (*Service)(nil)\n\nfunc (s *Service) MapCampaignNotFound(err error) error {\n\treturn mapNotFound(err, ErrCampaignNotFound)\n}\n\nvar _ privacyadmin.Host",
        "var _ opsadmin.BlacklistAdmin = (*Service)(nil)\n\nvar _ privacyadmin.Host",
    )
    out_path.write_text(merged)
    subprocess.run(["goimports", "-w", str(out_path)], check=True)

    merged_text = out_path.read_text()
    if "pkg/legal" not in merged_text:
        merged_text = merged_text.replace(
            '"ad-event-processor/pkg/platformconfig"',
            '"ad-event-processor/pkg/legal"\n\t"ad-event-processor/pkg/platformconfig"',
            1,
        )
        out_path.write_text(merged_text)
        subprocess.run(["goimports", "-w", str(out_path)], check=True)

    for name in MERGE_FILES:
        (ROOT / name).unlink()
    for name in EXTRA_DELETE:
        p = ROOT / name
        if p.exists():
            p.unlink()

    print(f"wrote {out_path}")
    backup = pathlib.Path(__file__).with_name("_admin_bridges_platform.go.bak")
    backup.write_text(out_path.read_text())
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
