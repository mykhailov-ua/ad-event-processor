package controlplane

import (
	"context"
	"fmt"
	"time"

	"ad-event-processor/internal/campaign"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/pkg/coldpath"

	"github.com/google/uuid"
)

var _ campaign.Effects = (*Service)(nil)

func (s *Service) CampaignFlowID(ctx context.Context, campaignID uuid.UUID) (string, error) {
	return s.campaignFlowID(ctx, campaignID)
}

func (s *Service) AttachCampaignBudgetApprovalState(ctx context.Context, dto *campaign.CampaignDTO) {
	s.attachCampaignBudgetApprovalState(ctx, dto)
}

func (s *Service) AttachCampaignListBudgetApprovalStates(ctx context.Context, items []campaign.CampaignDTO) {
	s.attachCampaignListBudgetApprovalStates(ctx, items)
}

func (s *Service) ApplyCampaignIngressCostPatch(ctx context.Context, campaignID uuid.UUID, cfg campaign.IngressCostConfigDTO) error {
	return s.applyCampaignIngressCostPatch(ctx, campaignID, cfg)
}

func (s *Service) ApplyCampaignClickPresetPatch(ctx context.Context, campaignID uuid.UUID, templateID *string, queryParams *map[string]string) error {
	return s.applyCampaignClickPresetPatch(ctx, campaignID, templateID, queryParams)
}

func (s *Service) ApplyCampaignBudgetPatch(ctx context.Context, q db.Querier, locked db.Campaign, budgetMicro int64) error {
	return s.applyCampaignBudgetPatch(ctx, q, locked, budgetMicro)
}

func (s *Service) ApplyCampaignSchedulePatch(ctx context.Context, q db.Querier, campaignID uuid.UUID, locked db.Campaign, startAt, endAt *time.Time, daypartHours []int16) error {
	return s.applyCampaignSchedulePatch(ctx, q, campaignID, locked, startAt, endAt, daypartHours)
}

func (s *Service) ApplyCampaignStatusPatch(ctx context.Context, q db.Querier, locked db.Campaign, status db.CampaignStatusType, reason string, publishForce bool) error {
	return s.applyCampaignStatusPatch(ctx, q, locked, status, reason, campaignStatusPatchOpts{publishForce: publishForce})
}

func (s *Service) ProxyAllowHTTPInsecure() bool {
	return s.cfg != nil && s.cfg.ProxyAllowHTTPInsecure
}

func (s *Service) PublishCampaignUpdate(ctx context.Context, campaignID string) {
	_ = s.publishCampaignUpdate(ctx, campaignID)
}

func (s *Service) ValidateFlowPaths(ctx context.Context, paths []campaign.FlowPathDTO) error {
	return s.validateFlowPaths(ctx, paths)
}

func (s *Service) EnforceCampaignPublishGate(ctx context.Context, campaignID uuid.UUID, row db.Campaign, force bool) error {
	return s.enforceCampaignPublishGate(ctx, campaignID, row, force)
}

func (s *Service) ResumeCampaignForPublish(ctx context.Context, campaignID uuid.UUID, force bool) error {
	return s.CampaignRuntime().ResumeCampaignForPublish(ctx, campaignID, force)
}

func (s *Service) EnqueueCampaignOutbox(ctx context.Context, q db.Querier, eventType string, campaignID uuid.UUID, budgetLimit int64) error {
	payload, err := coldpath.MarshalOutbox(CampaignPayload{CampaignID: campaignID.String(), BudgetLimit: budgetLimit})
	if err != nil {
		return fmt.Errorf("marshal %s outbox payload: %w", eventType, err)
	}
	_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{EventType: eventType, Payload: payload})
	return err
}

func (s *Service) AuditCampaignPublishForce(ctx context.Context, campaignID uuid.UUID) error {
	return s.auditCampaignPublishForce(ctx, campaignID)
}

func (s *Service) EnforceDeploymentLicenseCampaignCap(ctx context.Context) error {
	return s.enforceDeploymentLicenseCampaignCap(ctx)
}

func (s *Service) EmitCampaignLifecycleOutbox(ctx context.Context, q db.Querier, campaignID uuid.UUID, status db.CampaignStatusType, budgetLimit int64) error {
	return s.emitCampaignLifecycleOutbox(ctx, q, campaignID, status, budgetLimit)
}

func (s *Service) CampaignImportExportHost() campaign.ImportExportHost {
	return campaignImportExportHost{svc: s}
}

func (s *Service) CampaignRuntime() *campaign.Runtime {
	if s == nil {
		return nil
	}
	if s.campaignRuntime == nil {
		s.campaignRuntime = campaign.NewRuntime(s.pool, s)
		if s.clickhouseQuery != nil {
			s.campaignRuntime.SetClickHouseQuery(s.clickhouseQuery)
		}
	}
	return s.campaignRuntime
}
