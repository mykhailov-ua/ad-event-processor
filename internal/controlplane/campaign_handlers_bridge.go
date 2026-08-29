package controlplane

import (
	"context"
	"fmt"
	"time"

	"ad-event-processor/internal/campaign"
	"ad-event-processor/internal/campaign/runtime"
	"ad-event-processor/internal/database"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/flow"
	"ad-event-processor/internal/platformadmin"
	"ad-event-processor/pkg/coldpath"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func (s *Service) ApplyCampaignIngressCostPatch(ctx context.Context, campaignID uuid.UUID, cfg campaign.IngressCostConfigDTO) error {
	return campaign.ApplyIngressCostPatch(ctx, s, s.pool, campaignID, cfg)
}

func (s *Service) ApplyCampaignClickPresetPatch(ctx context.Context, campaignID uuid.UUID, templateID *string, queryParams *map[string]string) error {
	return campaign.ApplyClickPresetPatch(ctx, s, s.pool, campaignID, templateID, queryParams)
}

func (s *Service) ApplyCampaignBudgetPatch(ctx context.Context, q db.Querier, locked db.Campaign, budgetMicro int64) error {
	return campaign.ApplyBudgetPatch(ctx, s, q, locked, budgetMicro)
}

func (s *Service) HandleMediaBuyerBudgetIncrease(ctx context.Context, locked db.Campaign, userID uuid.UUID, newLimit int64) error {
	return platformadmin.HandleMediaBuyerBudgetIncrease(ctx, s, locked, userID, newLimit)
}

func (s *Service) ApplyCampaignSchedulePatch(ctx context.Context, q db.Querier, campaignID uuid.UUID, locked db.Campaign, startAt, endAt *time.Time, daypartHours []int16) error {
	return campaign.ApplySchedulePatch(ctx, s, q, campaignID, locked, startAt, endAt, daypartHours)
}

func (s *Service) ApplyCampaignStatusPatch(ctx context.Context, q db.Querier, locked db.Campaign, status db.CampaignStatusType, reason string, publishForce bool) error {
	return campaign.ApplyStatusPatch(ctx, s, q, locked, status, reason, publishForce)
}

func (s *Service) CloneCampaignPlacementBlocks(ctx context.Context, sourceID, destID uuid.UUID) error {
	return s.cloneCampaignPlacementBlocks(ctx, sourceID, destID)
}

func (s *Service) CloneCampaign(ctx context.Context, spec campaign.CloneCampaignSpec) (campaign.CloneCampaignResult, error) {
	return campaign.CloneCampaign(ctx, s, s.GetPool(), spec)
}

func (s *Service) ProxyAllowHTTPInsecure() bool {
	return s.cfg != nil && s.cfg.ProxyAllowHTTPInsecure
}

func (s *Service) PublishCampaignUpdate(ctx context.Context, campaignID string) {
	_ = s.publishCampaignUpdate(ctx, campaignID)
}

func (s *Service) ValidateFlowPaths(ctx context.Context, paths []flow.PathDTO) error {
	return flow.ValidatePathRefs(ctx, s, paths)
}

func (s *Service) EnforceCampaignPublishGate(ctx context.Context, campaignID uuid.UUID, row db.Campaign, force bool) error {
	return campaign.EnforcePublishGate(ctx, s, s.GetPool(), campaignID, row, force)
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
	return campaign.AuditPublishForce(ctx, s, s.GetPool(), campaignID)
}

func (s *Service) EnforceDeploymentLicenseCampaignCap(ctx context.Context) error {
	return s.enforceDeploymentLicenseCampaignCap(ctx)
}

func (s *Service) AssignCampaignBrand(ctx context.Context, campaignID, brandID uuid.UUID) error {
	return campaign.AssignCampaignBrand(ctx, s, s.pool, campaignID, brandID)
}

func (s *Service) EmitCampaignLifecycleOutbox(ctx context.Context, q db.Querier, campaignID uuid.UUID, status db.CampaignStatusType, budgetLimit int64) error {
	return campaign.EmitCampaignLifecycleOutbox(ctx, q, campaignID, status, budgetLimit)
}

func (s *Service) CampaignImportExportHost() campaign.ImportExportHost {
	return campaignImportExportHost{svc: s}
}

func (s *Service) CampaignRuntime() *runtime.Runtime {
	if s == nil {
		return nil
	}
	if s.campaignRuntime == nil {
		s.campaignRuntime = runtime.NewRuntime(s.pool, s)
		if s.clickhouseQuery != nil {
			s.campaignRuntime.SetClickHouseQuery(s.clickhouseQuery)
		}
	}
	return s.campaignRuntime
}

func formatOptionalText(t pgtype.Text) string {
	return campaign.FormatOptionalText(t)
}

func (s *Service) ListCampaigns(ctx context.Context, customerID uuid.UUID, status string, limit, offset int32) ([]campaign.CampaignDTO, int64, error) {
	return s.CampaignRuntime().ListCampaigns(ctx, customerID, status, limit, offset)
}

func (s *Service) GetCampaign(ctx context.Context, id uuid.UUID) (campaign.CampaignDTO, error) {
	return s.CampaignRuntime().GetCampaign(ctx, id)
}

func (s *Service) PatchCampaign(ctx context.Context, campaignID uuid.UUID, req campaign.PatchCampaignRequest) (campaign.CampaignDTO, error) {
	return s.CampaignRuntime().PatchCampaign(ctx, campaignID, req)
}

func (s *Service) ListCampaignEvents(ctx context.Context, campaignID uuid.UUID, limit, offset int32) ([]campaign.CampaignEventDTO, int64, error) {
	return s.CampaignRuntime().ListCampaignEvents(ctx, campaignID, limit, offset)
}

func (s *Service) ListStatusHistory(ctx context.Context, campaignID uuid.UUID, limit, offset int32) ([]campaign.StatusHistoryDTO, int64, error) {
	return s.CampaignRuntime().ListStatusHistory(ctx, campaignID, limit, offset)
}

func (s *Service) SetClickHouse(conn driver.Conn, cfg database.ClickHouseQueryConfig) {
	if conn != nil {
		s.clickhouseQuery = database.NewClickHouseQuery(conn, cfg)
		if cr := s.campaignRuntime; cr != nil {
			cr.SetClickHouseQuery(s.clickhouseQuery)
		}
	}
}

func (s *Service) SetClickHouseWrite(conn driver.Conn) {
	s.clickhouseWriteConn = conn
}

func (s *Service) UpdateCampaignPacing(ctx context.Context, campaignID uuid.UUID, newMode string) (campaign.CampaignDTO, error) {
	return s.CampaignRuntime().UpdateCampaignPacing(ctx, campaignID, newMode)
}

func (s *Service) GetCampaignStats(ctx context.Context, campaignID uuid.UUID, from, to time.Time, granularity string) (campaign.CampaignStatsDTO, error) {
	return s.CampaignRuntime().GetCampaignStats(ctx, campaignID, from, to, granularity)
}

func (s *Service) CreateCampaign(ctx context.Context, spec campaign.CreateCampaignSpec) (uuid.UUID, error) {
	return s.CampaignRuntime().CreateCampaign(ctx, spec)
}

func (s *Service) PauseCampaign(ctx context.Context, campaignID uuid.UUID, reason string) error {
	return s.CampaignRuntime().PauseCampaign(ctx, campaignID, reason)
}

func (s *Service) PreviewPauseCampaign(ctx context.Context, campaignID uuid.UUID, reason string) (campaign.MutationPreviewDTO, error) {
	return campaign.PreviewPauseCampaign(ctx, s.pool, campaignID, reason)
}

func (s *Service) ResumeCampaign(ctx context.Context, campaignID uuid.UUID, reason string) error {
	return s.CampaignRuntime().ResumeCampaign(ctx, campaignID, reason)
}

func (s *Service) PreviewResumeCampaign(ctx context.Context, campaignID uuid.UUID, reason string) (campaign.MutationPreviewDTO, error) {
	return campaign.PreviewResumeCampaign(ctx, s.pool, s, campaignID, reason)
}

func (s *Service) UpdateCampaignSchedule(ctx context.Context, campaignID uuid.UUID, startAt, endAt *time.Time, daypartHours []int16) error {
	return s.CampaignRuntime().UpdateCampaignSchedule(ctx, campaignID, startAt, endAt, daypartHours)
}

func (s *Service) clickHouseIngestionLag(ctx context.Context) (time.Duration, error) {
	return runtime.ClickHouseIngestionLag(ctx, s.clickhouseQuery)
}

func (s *Service) RunCampaignSmoke(ctx context.Context, campaignID uuid.UUID) (campaign.CampaignSmokeResultDTO, error) {
	return campaign.RunCampaignSmoke(ctx, s, campaignID)
}

func (s *Service) SmokeServiceAvailable() bool {
	return s != nil && s.pool != nil
}
