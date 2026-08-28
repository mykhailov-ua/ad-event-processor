package campaign

import (
	"context"
	"time"

	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
)

type Effects interface {
	CampaignFlowID(ctx context.Context, campaignID uuid.UUID) (string, error)
	AttachCampaignBudgetApprovalState(ctx context.Context, dto *CampaignDTO)
	AttachCampaignListBudgetApprovalStates(ctx context.Context, items []CampaignDTO)
	GetCampaignMargin(ctx context.Context, campaignID uuid.UUID) (CampaignMarginDTO, error)
	AttachCampaignListMarginBreach(ctx context.Context, items []CampaignDTO)

	GetCampaignRow(ctx context.Context, campaignID uuid.UUID) (db.Campaign, error)
	AssignCampaignFlow(ctx context.Context, campaignID, flowID uuid.UUID) error
	ApplyCampaignIngressCostPatch(ctx context.Context, campaignID uuid.UUID, cfg IngressCostConfigDTO) error
	ApplyCampaignClickPresetPatch(ctx context.Context, campaignID uuid.UUID, templateID *string, queryParams *map[string]string) error
	AssignCampaignBrand(ctx context.Context, campaignID, brandID uuid.UUID) error
	ApplyCampaignBudgetPatch(ctx context.Context, q db.Querier, locked db.Campaign, budgetMicro int64) error
	ApplyCampaignSchedulePatch(ctx context.Context, q db.Querier, campaignID uuid.UUID, locked db.Campaign, startAt, endAt *time.Time, daypartHours []int16) error
	ApplyCampaignStatusPatch(ctx context.Context, q db.Querier, locked db.Campaign, status db.CampaignStatusType, reason string, publishForce bool) error
	ProxyAllowHTTPInsecure() bool
	PublishCampaignUpdate(ctx context.Context, campaignID string)
	AuditLog(ctx context.Context, q db.Querier, adminID uuid.UUID, action, targetType string, targetID *uuid.UUID, changes, metadata any)

	GetFlow(ctx context.Context, flowID uuid.UUID) (FlowDTO, error)
	ValidateFlowPaths(ctx context.Context, paths []FlowPathDTO) error
	GetCampaignIntegrationHealth(ctx context.Context, campaignID uuid.UUID) (IntegrationHealthDTO, error)
	EnforceCampaignPublishGate(ctx context.Context, campaignID uuid.UUID, row db.Campaign, force bool) error
	ResumeCampaignForPublish(ctx context.Context, campaignID uuid.UUID, force bool) error
	AuditCampaignPublishForce(ctx context.Context, campaignID uuid.UUID) error

	AssignCampaignOwner(ctx context.Context, campaignID, ownerUserID uuid.UUID) error
	BlockCampaignPlacement(ctx context.Context, campaignID uuid.UUID, placementID string) error
	CloneCampaign(ctx context.Context, spec CloneCampaignSpec) (CloneCampaignResult, error)
	ImportMigrationCampaigns(ctx context.Context, spec ImportMigrationSpec) (ImportMigrationResult, error)
	EnqueueCampaignOutbox(ctx context.Context, q db.Querier, eventType string, campaignID uuid.UUID, budgetLimit int64) error
	EnforceDeploymentLicenseCampaignCap(ctx context.Context) error
	EmitCampaignLifecycleOutbox(ctx context.Context, q db.Querier, campaignID uuid.UUID, status db.CampaignStatusType, budgetLimit int64) error
	CampaignImportExportHost() ImportExportHost
}

func errServiceUnavailable() error {
	return errValidation("service unavailable")
}
