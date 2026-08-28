package controlplane

import (
	"context"

	"ad-event-processor/internal/campaign"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const campaignExportVersion = campaign.CampaignExportVersion

type campaignImportExportHost struct {
	svc *Service
}

func (h campaignImportExportHost) Pool() *pgxpool.Pool {
	return h.svc.GetPool()
}

func (h campaignImportExportHost) AssertMediaBuyerCampaignAccess(ctx context.Context, row db.Campaign) error {
	return assertMediaBuyerCampaignAccess(ctx, row)
}

func (h campaignImportExportHost) GetFlow(ctx context.Context, flowID uuid.UUID) (FlowDTO, error) {
	return h.svc.GetFlow(ctx, flowID)
}

func (h campaignImportExportHost) AuditImportCampaign(ctx context.Context, q *db.Queries, campaignID uuid.UUID, change campaign.ImportCampaignAuditChange, meta campaign.ImportCampaignIdempotencyMeta) error {
	h.svc.AuditLog(ctx, q, uuid.Nil, "IMPORT_CAMPAIGN", "campaign", &campaignID, auditImportCampaignChange{Name: change.Name}, auditIdempotencyMeta{IdempotencyKey: meta.IdempotencyKey})
	return nil
}

func (h campaignImportExportHost) EmitCampaignLifecycleOutbox(ctx context.Context, q *db.Queries, campaignID uuid.UUID, status db.CampaignStatusType, budget int64) error {
	return h.svc.emitCampaignLifecycleOutbox(ctx, q, campaignID, status, budget)
}

func (h campaignImportExportHost) PublishCampaignUpdate(ctx context.Context, campaignID string) {
	_ = h.svc.publishCampaignUpdate(ctx, campaignID)
}

func (h campaignImportExportHost) PublishFlowReload(ctx context.Context) {
	_ = h.svc.PublishFlowReload(ctx)
}

func (s *Service) ExportCampaign(ctx context.Context, campaignID uuid.UUID) (CampaignExportBundle, error) {
	return s.CampaignRuntime().ExportCampaign(ctx, campaignID)
}

func (s *Service) ImportCampaign(ctx context.Context, spec ImportCampaignSpec) (ImportCampaignResult, error) {
	return s.CampaignRuntime().ImportCampaign(ctx, spec)
}

type auditImportCampaignChange struct {
	Name string `json:"name"`
}
