package campaign

import (
	"context"
	"time"

	"ad-event-processor/internal/database"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Runtime struct {
	pool            *pgxpool.Pool
	effects         Effects
	clickhouseQuery *database.ClickHouseQuery
}

func NewRuntime(pool *pgxpool.Pool, effects Effects) *Runtime {
	return &Runtime{pool: pool, effects: effects}
}

func (r *Runtime) SetClickHouseQuery(q *database.ClickHouseQuery) {
	if r == nil {
		return
	}
	r.clickhouseQuery = q
}

func (r *Runtime) poolOrNil() *pgxpool.Pool {
	if r == nil {
		return nil
	}
	return r.pool
}

func (r *Runtime) ListCampaigns(ctx context.Context, customerID uuid.UUID, status string, limit, offset int32) ([]CampaignDTO, int64, error) {
	return listCampaigns(ctx, r.poolOrNil(), r.effects, customerID, status, limit, offset)
}

func (r *Runtime) GetCampaign(ctx context.Context, campaignID uuid.UUID) (CampaignDTO, error) {
	return getCampaign(ctx, r.poolOrNil(), r.effects, campaignID)
}

func (r *Runtime) GetCampaignMargin(ctx context.Context, campaignID uuid.UUID) (CampaignMarginDTO, error) {
	if r == nil || r.effects == nil {
		return CampaignMarginDTO{}, errServiceUnavailable()
	}
	return r.effects.GetCampaignMargin(ctx, campaignID)
}

func (r *Runtime) AttachCampaignListMarginBreach(ctx context.Context, items []CampaignDTO) {
	if r == nil || r.effects == nil {
		return
	}
	r.effects.AttachCampaignListMarginBreach(ctx, items)
}

func (r *Runtime) PatchCampaign(ctx context.Context, campaignID uuid.UUID, req PatchCampaignRequest) (CampaignDTO, error) {
	if r == nil || r.effects == nil {
		return CampaignDTO{}, errServiceUnavailable()
	}
	return patchCampaign(ctx, r.poolOrNil(), r.effects, campaignID, req)
}

func (r *Runtime) PublishCampaign(ctx context.Context, campaignID uuid.UUID, force bool) (CampaignDTO, error) {
	if r == nil || r.effects == nil {
		return CampaignDTO{}, errServiceUnavailable()
	}
	return publishCampaign(ctx, r.poolOrNil(), r.effects, campaignID, force)
}

func (r *Runtime) EvaluateCampaignPublish(ctx context.Context, campaignID uuid.UUID) (CampaignPublishCheckDTO, error) {
	if r == nil || r.effects == nil {
		return CampaignPublishCheckDTO{}, errServiceUnavailable()
	}
	return evaluateCampaignPublish(ctx, r.effects, campaignID)
}

func (r *Runtime) AssignCampaignOwner(ctx context.Context, campaignID, ownerUserID uuid.UUID) error {
	if r == nil || r.effects == nil {
		return errServiceUnavailable()
	}
	return r.effects.AssignCampaignOwner(ctx, campaignID, ownerUserID)
}

func (r *Runtime) ListCampaignEvents(ctx context.Context, campaignID uuid.UUID, limit, offset int32) ([]CampaignEventDTO, int64, error) {
	return listCampaignEvents(ctx, r.poolOrNil(), campaignID, limit, offset)
}

func (r *Runtime) ListStatusHistory(ctx context.Context, campaignID uuid.UUID, limit, offset int32) ([]StatusHistoryDTO, int64, error) {
	return listStatusHistory(ctx, r.poolOrNil(), campaignID, limit, offset)
}

func (r *Runtime) BlockCampaignPlacement(ctx context.Context, campaignID uuid.UUID, placementID string) error {
	if r == nil || r.effects == nil {
		return errServiceUnavailable()
	}
	return r.effects.BlockCampaignPlacement(ctx, campaignID, placementID)
}

func (r *Runtime) CloneCampaign(ctx context.Context, spec CloneCampaignSpec) (CloneCampaignResult, error) {
	if r == nil || r.effects == nil {
		return CloneCampaignResult{}, errServiceUnavailable()
	}
	return r.effects.CloneCampaign(ctx, spec)
}

func (r *Runtime) ExportCampaign(ctx context.Context, campaignID uuid.UUID) (CampaignExportBundle, error) {
	if r == nil || r.effects == nil {
		return CampaignExportBundle{}, errServiceUnavailable()
	}
	return ExportCampaign(ctx, r.effects.CampaignImportExportHost(), campaignID)
}

func (r *Runtime) ImportCampaign(ctx context.Context, spec ImportCampaignSpec) (ImportCampaignResult, error) {
	if r == nil || r.effects == nil {
		return ImportCampaignResult{}, errServiceUnavailable()
	}
	return ImportCampaign(ctx, r.effects.CampaignImportExportHost(), spec)
}

func (r *Runtime) ImportMigrationCampaigns(ctx context.Context, spec ImportMigrationSpec) (ImportMigrationResult, error) {
	if r == nil || r.effects == nil {
		return ImportMigrationResult{}, errServiceUnavailable()
	}
	return r.effects.ImportMigrationCampaigns(ctx, spec)
}

func (r *Runtime) GetCampaignIntegrationHealth(ctx context.Context, campaignID uuid.UUID) (IntegrationHealthDTO, error) {
	if r == nil || r.effects == nil {
		return IntegrationHealthDTO{}, errServiceUnavailable()
	}
	return r.effects.GetCampaignIntegrationHealth(ctx, campaignID)
}

func (r *Runtime) CreateCampaign(ctx context.Context, spec CreateCampaignSpec) (uuid.UUID, error) {
	if r == nil || r.effects == nil {
		return uuid.Nil, errServiceUnavailable()
	}
	return createCampaign(ctx, r.poolOrNil(), r.effects, spec)
}

func (r *Runtime) UpdateCampaignSchedule(ctx context.Context, campaignID uuid.UUID, startAt, endAt *time.Time, daypartHours []int16) error {
	if r == nil || r.effects == nil {
		return errServiceUnavailable()
	}
	return updateCampaignSchedule(ctx, r.poolOrNil(), r.effects, campaignID, startAt, endAt, daypartHours)
}

func (r *Runtime) UpdateCampaignPacing(ctx context.Context, campaignID uuid.UUID, newMode string) (CampaignDTO, error) {
	if r == nil || r.effects == nil {
		return CampaignDTO{}, errServiceUnavailable()
	}
	return updateCampaignPacing(ctx, r.poolOrNil(), r.effects, campaignID, newMode)
}

func (r *Runtime) GetCampaignStats(ctx context.Context, campaignID uuid.UUID, from, to time.Time, granularity string) (CampaignStatsDTO, error) {
	if r == nil {
		return CampaignStatsDTO{}, errServiceUnavailable()
	}
	return getCampaignStats(ctx, r.poolOrNil(), r.clickhouseQuery, campaignID, from, to, granularity)
}

func (r *Runtime) PauseCampaign(ctx context.Context, campaignID uuid.UUID, reason string) error {
	if r == nil || r.effects == nil {
		return errServiceUnavailable()
	}
	return pauseCampaign(ctx, r.poolOrNil(), r.effects, campaignID, reason)
}

func (r *Runtime) ResumeCampaign(ctx context.Context, campaignID uuid.UUID, reason string) error {
	if r == nil || r.effects == nil {
		return errServiceUnavailable()
	}
	return resumeCampaign(ctx, r.poolOrNil(), r.effects, campaignID, reason, false)
}

func (r *Runtime) ResumeCampaignForPublish(ctx context.Context, campaignID uuid.UUID, force bool) error {
	if r == nil || r.effects == nil {
		return errServiceUnavailable()
	}
	return resumeCampaign(ctx, r.poolOrNil(), r.effects, campaignID, "publish", force)
}

var _ CampaignReader = (*Runtime)(nil)
