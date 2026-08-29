package campaign

import (
	"context"
	"time"

	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/flow"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ReadHost interface {
	GetCampaignRow(ctx context.Context, campaignID uuid.UUID) (db.Campaign, error)
	ProxyAllowHTTPInsecure() bool
}

type IntegrationHost interface {
	CampaignFlowID(ctx context.Context, campaignID uuid.UUID) (string, error)
	AttachCampaignBudgetApprovalState(ctx context.Context, dto *CampaignDTO)
	AttachCampaignListBudgetApprovalStates(ctx context.Context, items []CampaignDTO)
	GetCampaignMargin(ctx context.Context, campaignID uuid.UUID) (CampaignMarginDTO, error)
	AttachCampaignListMarginBreach(ctx context.Context, items []CampaignDTO)
	AssignCampaignFlow(ctx context.Context, campaignID, flowID uuid.UUID) error
	GetFlow(ctx context.Context, flowID uuid.UUID) (flow.DTO, error)
	ValidateFlowPaths(ctx context.Context, paths []flow.PathDTO) error
	GetCampaignIntegrationHealth(ctx context.Context, campaignID uuid.UUID) (IntegrationHealthDTO, error)
}

type PatchHost interface {
	ApplyCampaignIngressCostPatch(ctx context.Context, campaignID uuid.UUID, cfg IngressCostConfigDTO) error
	ApplyCampaignClickPresetPatch(ctx context.Context, campaignID uuid.UUID, templateID *string, queryParams *map[string]string) error
	AssignCampaignBrand(ctx context.Context, campaignID, brandID uuid.UUID) error
	ApplyCampaignBudgetPatch(ctx context.Context, q db.Querier, locked db.Campaign, budgetMicro int64) error
	ApplyCampaignSchedulePatch(ctx context.Context, q db.Querier, campaignID uuid.UUID, locked db.Campaign, startAt, endAt *time.Time, daypartHours []int16) error
	ApplyCampaignStatusPatch(ctx context.Context, q db.Querier, locked db.Campaign, status db.CampaignStatusType, reason string, publishForce bool) error
	HandleMediaBuyerBudgetIncrease(ctx context.Context, locked db.Campaign, userID uuid.UUID, newLimit int64) error
}

type PublishHost interface {
	EnforceCampaignPublishGate(ctx context.Context, campaignID uuid.UUID, row db.Campaign, force bool) error
	ResumeCampaignForPublish(ctx context.Context, campaignID uuid.UUID, force bool) error
}

type CloneHost interface {
	CloneCampaign(ctx context.Context, spec CloneCampaignSpec) (CloneCampaignResult, error)
	CloneCampaignPlacementBlocks(ctx context.Context, sourceID, destID uuid.UUID) error
	AssignCampaignOwner(ctx context.Context, campaignID, ownerUserID uuid.UUID) error
	BlockCampaignPlacement(ctx context.Context, campaignID uuid.UUID, placementID string) error
}

type MutationNotifyHost interface {
	PublishCampaignUpdate(ctx context.Context, campaignID string)
	PublishFlowReload(ctx context.Context) error
	EnqueueCampaignOutbox(ctx context.Context, q db.Querier, eventType string, campaignID uuid.UUID, budgetLimit int64) error
	EmitCampaignLifecycleOutbox(ctx context.Context, q db.Querier, campaignID uuid.UUID, status db.CampaignStatusType, budgetLimit int64) error
	AuditLog(ctx context.Context, q db.Querier, adminID uuid.UUID, action, targetType string, targetID *uuid.UUID, changes, metadata any)
}

type MigrationHost interface {
	ImportMigrationCampaigns(ctx context.Context, spec ImportMigrationSpec) (ImportMigrationResult, error)
	ImportCampaign(ctx context.Context, spec ImportCampaignSpec) (ImportCampaignResult, error)
	CampaignImportExportHost() ImportExportHost
	EnforceDeploymentLicenseCampaignCap(ctx context.Context) error
}

type ImportExportHost interface {
	Pool() *pgxpool.Pool
	AssertMediaBuyerCampaignAccess(ctx context.Context, row db.Campaign) error
	GetFlow(ctx context.Context, flowID uuid.UUID) (FlowDTO, error)
	AuditImportCampaign(ctx context.Context, q *db.Queries, campaignID uuid.UUID, change ImportCampaignAuditChange, meta ImportCampaignIdempotencyMeta) error
	EmitCampaignLifecycleOutbox(ctx context.Context, q *db.Queries, campaignID uuid.UUID, status db.CampaignStatusType, budget int64) error
	PublishCampaignUpdate(ctx context.Context, campaignID string)
	PublishFlowReload(ctx context.Context)
}

type ImportCampaignAuditChange struct {
	Name string `json:"name"`
}

type ImportCampaignIdempotencyMeta struct {
	IdempotencyKey string `json:"idempotency_key"`
}

type Effects interface {
	ReadHost
	IntegrationHost
	PatchHost
	PublishHost
	CloneHost
	MutationNotifyHost
	MigrationHost
}

type ListEffects interface {
	IntegrationHost
}

type PublishGateEffects interface {
	ReadHost
	IntegrationHost
}

type PatchEffects interface {
	ReadHost
	IntegrationHost
	PatchHost
	MutationNotifyHost
	PublishHost
}

type LifecycleEffects interface {
	ReadHost
	PublishHost
	MutationNotifyHost
}

type CloneEffects interface {
	MutationNotifyHost
	CloneHost
}

type CreateEffects interface {
	MigrationHost
	MutationNotifyHost
}

type MigrationPullEffects interface {
	MigrationHost
}

type MigrationImportEffects interface {
	MigrationHost
	MutationNotifyHost
}

type AssignBrandEffects interface {
	ReadHost
	MutationNotifyHost
}

type IntegrationHealthEffects interface {
	ReadHost
	IntegrationHost
	MutationNotifyHost
}

type BudgetPatchEffects interface {
	PatchHost
	MutationNotifyHost
}

type StatusPatchEffects interface {
	PublishHost
	MutationNotifyHost
}

type PacingUpdateEffects interface {
	MutationNotifyHost
}

type ScheduleUpdateEffects interface {
	PatchHost
	PublishHost
}

type ClickPresetPatchEffects interface {
	ReadHost
	MutationNotifyHost
}

type previewResumeEffects interface {
	PublishHost
}

type schedulePatchEffects interface {
	MutationNotifyHost
	PublishHost
}

type publishCampaignEffects interface {
	PublishGateEffects
	PublishHost
	IntegrationHost
}

type enforcePublishGateEffects interface {
	PublishGateEffects
	MutationNotifyHost
}
