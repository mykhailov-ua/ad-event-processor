package controlplane

import (
	"ad-event-processor/internal/automation"
	"ad-event-processor/internal/campaign"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/flow"
	"ad-event-processor/internal/licensing"
	"ad-event-processor/internal/migrationsource"
	"ad-event-processor/internal/outbox"
	"ad-event-processor/internal/platformadmin"
	"ad-event-processor/pkg/coldpath"
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func (s *Service) AutoscaleBudgets(ctx context.Context, syncWorkers []*domain.SyncWorker) error {
	return campaign.RunAutoscaleBudgetsTick(ctx, s, syncWorkers)
}

type campaignExperimentsHost struct {
	svc *Service
}

func (h campaignExperimentsHost) Pool() *pgxpool.Pool {
	return h.svc.GetPool()
}

func (h campaignExperimentsHost) CohortSnapshotOutboxPayload() ([]byte, error) {
	return coldpath.MarshalOutbox(outbox.CohortSnapshotPayload{Version: 1})
}

func (h campaignExperimentsHost) AuditCohortSnapshotChange(ctx context.Context, q db.Querier, experimentID uuid.UUID, change campaign.ExperimentCohortAuditChange, outboxEventID int64) {
	h.svc.AuditLog(ctx, q, uuid.Nil, "UPDATE_COHORT_SNAPSHOT", "experiment", &experimentID, platformadmin.AuditCohortSnapshotChange{
		Name:     change.Name,
		Active:   change.Active,
		Variants: change.Variants,
	}, platformadmin.AuditOutboxEventMeta{OutboxEventID: outboxEventID})
}

func (s *Service) UpsertExperimentCohort(ctx context.Context, spec campaign.ExperimentCohortSpec) error {
	return campaign.UpsertExperimentCohort(ctx, campaignExperimentsHost{svc: s}, spec)
}

type campaignImportExportHost struct {
	svc *Service
}

func (h campaignImportExportHost) Pool() *pgxpool.Pool {
	return h.svc.GetPool()
}

func (h campaignImportExportHost) AssertMediaBuyerCampaignAccess(ctx context.Context, row db.Campaign) error {
	return assertMediaBuyerCampaignAccess(ctx, row)
}

func (h campaignImportExportHost) GetFlow(ctx context.Context, flowID uuid.UUID) (flow.DTO, error) {
	return h.svc.GetFlow(ctx, flowID)
}

func (h campaignImportExportHost) AuditImportCampaign(ctx context.Context, q *db.Queries, campaignID uuid.UUID, change campaign.ImportCampaignAuditChange, meta campaign.ImportCampaignIdempotencyMeta) error {
	h.svc.AuditLog(ctx, q, uuid.Nil, "IMPORT_CAMPAIGN", "campaign", &campaignID, auditImportCampaignChange{Name: change.Name}, platformadmin.AuditIdempotencyMeta{IdempotencyKey: meta.IdempotencyKey})
	return nil
}

func (h campaignImportExportHost) EmitCampaignLifecycleOutbox(ctx context.Context, q *db.Queries, campaignID uuid.UUID, status db.CampaignStatusType, budget int64) error {
	return h.svc.EmitCampaignLifecycleOutbox(ctx, q, campaignID, status, budget)
}

func (h campaignImportExportHost) PublishCampaignUpdate(ctx context.Context, campaignID string) {
	_ = h.svc.publishCampaignUpdate(ctx, campaignID)
}

func (h campaignImportExportHost) PublishFlowReload(ctx context.Context) {
	_ = h.svc.PublishFlowReload(ctx)
}

func (s *Service) ExportCampaign(ctx context.Context, campaignID uuid.UUID) (campaign.CampaignExportBundle, error) {
	return s.CampaignRuntime().ExportCampaign(ctx, campaignID)
}

func (s *Service) ImportCampaign(ctx context.Context, spec campaign.ImportCampaignSpec) (campaign.ImportCampaignResult, error) {
	return s.CampaignRuntime().ImportCampaign(ctx, spec)
}

func (s *Service) ImportMigrationCampaigns(ctx context.Context, spec campaign.ImportMigrationSpec) (campaign.ImportMigrationResult, error) {
	return campaign.ImportMigrationCampaigns(ctx, s, spec)
}

func (s *Service) PreviewMigrationPull(ctx context.Context, spec campaign.PullMigrationPreviewSpec) (migrationsource.PreviewResult, error) {
	return campaign.PreviewMigrationPull(ctx, s, spec)
}

func (s *Service) ImportMigrationPull(ctx context.Context, spec campaign.PullMigrationImportSpec) (campaign.ImportMigrationResult, error) {
	return campaign.ImportMigrationPull(ctx, s, spec)
}

type auditImportCampaignChange struct {
	Name string `json:"name"`
}

func (s *Service) PostbackEncryptionKey() []byte {
	return []byte("postback-encryption-secret-key32")
}

func (s *Service) TemplateCatalog(pool *pgxpool.Pool) *campaign.TemplateCatalog {
	if s == nil {
		return nil
	}
	if pool == nil {
		pool = s.pool
	}
	return campaign.NewTemplateCatalog(pool, s)
}

func (s *Service) ApplyCampaignTemplates(ctx context.Context, campaignID uuid.UUID, req campaign.ApplyCampaignTemplatesRequest) (campaign.ApplyCampaignTemplatesResult, error) {
	return s.TemplateCatalog(s.pool).ApplyCampaignTemplates(ctx, campaignID, req)
}

func (s *Service) AutomationRules() *automation.RulesService {
	if s == nil {
		return nil
	}
	return &automation.RulesService{
		Pool:       s.pool,
		ClickHouse: s.ClickHouseQuery(),
		EvalFloorMinutes: func() int {
			if s.cfg == nil {
				return 15
			}
			return s.cfg.Management.AutomationRulesIntervalMin
		},
		LicenseGate: s,
	}
}

func (s *Service) ValidateAutomationLicense(ctx context.Context, actions []automation.Action) error {
	for _, action := range actions {
		if action.Type != automation.ActionPlatformPause {
			continue
		}
		snap, err := licensing.LoadDeploymentSnapshot(ctx, s.pool)
		if err != nil || !snap.ModuleAllowed(func(f licensing.FeatureSet) bool { return f.AdPlatformCampaignAPIEnabled() }) {
			return fmt.Errorf("platform_pause requires ad_platform_campaign_api license")
		}
	}
	return nil
}

func (s *Service) CampaignFlowID(ctx context.Context, campaignID uuid.UUID) (string, error) {
	return s.campaignFlowID(ctx, campaignID)
}

func (s *Service) AttachCampaignBudgetApprovalState(ctx context.Context, dto *campaign.CampaignDTO) {
	platformadmin.AttachCampaignBudgetApprovalState(ctx, s, dto)
}

func (s *Service) AttachCampaignListBudgetApprovalStates(ctx context.Context, items []campaign.CampaignDTO) {
	platformadmin.AttachCampaignListBudgetApprovalStates(ctx, s, items)
}
