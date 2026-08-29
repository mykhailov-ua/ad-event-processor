package runtime

import (
	"context"
	"time"

	"ad-event-processor/internal/campaign"
	"ad-event-processor/internal/campaign/importexport"

	"ad-event-processor/internal/database"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Runtime struct {
	pool            *pgxpool.Pool
	effects         campaign.Effects
	clickhouseQuery *database.ClickHouseQuery
}

func NewRuntime(pool *pgxpool.Pool, effects campaign.Effects) *Runtime {
	return &Runtime{pool: pool, effects: effects}
}

func (r *Runtime) SetClickHouseQuery(q *database.ClickHouseQuery) {
	if r == nil {
		return
	}
	r.clickhouseQuery = q
}

func (r *Runtime) PoolOrNil() *pgxpool.Pool {
	if r == nil {
		return nil
	}
	return r.pool
}

func (r *Runtime) ListCampaigns(ctx context.Context, customerID uuid.UUID, status string, limit, offset int32) ([]campaign.CampaignDTO, int64, error) {
	return listCampaigns(ctx, r.PoolOrNil(), r.effects, customerID, status, limit, offset)
}

func (r *Runtime) GetCampaign(ctx context.Context, campaignID uuid.UUID) (campaign.CampaignDTO, error) {
	return getCampaign(ctx, r.PoolOrNil(), r.effects, campaignID)
}

func (r *Runtime) GetCampaignMargin(ctx context.Context, campaignID uuid.UUID) (campaign.CampaignMarginDTO, error) {
	if r == nil || r.effects == nil {
		return campaign.CampaignMarginDTO{}, campaign.ErrServiceUnavailable()
	}
	return r.effects.GetCampaignMargin(ctx, campaignID)
}

func (r *Runtime) AttachCampaignListMarginBreach(ctx context.Context, items []campaign.CampaignDTO) {
	if r == nil || r.effects == nil {
		return
	}
	r.effects.AttachCampaignListMarginBreach(ctx, items)
}

func (r *Runtime) PatchCampaign(ctx context.Context, campaignID uuid.UUID, req campaign.PatchCampaignRequest) (campaign.CampaignDTO, error) {
	if r == nil || r.effects == nil {
		return campaign.CampaignDTO{}, campaign.ErrServiceUnavailable()
	}
	return patchCampaign(ctx, r.PoolOrNil(), r.effects, campaignID, req)
}

func (r *Runtime) PublishCampaign(ctx context.Context, campaignID uuid.UUID, force bool) (campaign.CampaignDTO, error) {
	if r == nil || r.effects == nil {
		return campaign.CampaignDTO{}, campaign.ErrServiceUnavailable()
	}
	return publishCampaign(ctx, r.PoolOrNil(), r.effects, campaignID, force)
}

func (r *Runtime) EvaluateCampaignPublish(ctx context.Context, campaignID uuid.UUID) (campaign.CampaignPublishCheckDTO, error) {
	if r == nil || r.effects == nil {
		return campaign.CampaignPublishCheckDTO{}, campaign.ErrServiceUnavailable()
	}
	return evaluateCampaignPublish(ctx, r.effects, campaignID)
}

func (r *Runtime) AssignCampaignOwner(ctx context.Context, campaignID, ownerUserID uuid.UUID) error {
	if r == nil || r.effects == nil {
		return campaign.ErrServiceUnavailable()
	}
	return r.effects.AssignCampaignOwner(ctx, campaignID, ownerUserID)
}

func (r *Runtime) ListCampaignEvents(ctx context.Context, campaignID uuid.UUID, limit, offset int32) ([]campaign.CampaignEventDTO, int64, error) {
	return listCampaignEvents(ctx, r.PoolOrNil(), campaignID, limit, offset)
}

func (r *Runtime) ListStatusHistory(ctx context.Context, campaignID uuid.UUID, limit, offset int32) ([]campaign.StatusHistoryDTO, int64, error) {
	return listStatusHistory(ctx, r.PoolOrNil(), campaignID, limit, offset)
}

func (r *Runtime) BlockCampaignPlacement(ctx context.Context, campaignID uuid.UUID, placementID string) error {
	if r == nil || r.effects == nil {
		return campaign.ErrServiceUnavailable()
	}
	return r.effects.BlockCampaignPlacement(ctx, campaignID, placementID)
}

func (r *Runtime) CloneCampaign(ctx context.Context, spec campaign.CloneCampaignSpec) (campaign.CloneCampaignResult, error) {
	if r == nil || r.effects == nil {
		return campaign.CloneCampaignResult{}, campaign.ErrServiceUnavailable()
	}
	return r.effects.CloneCampaign(ctx, spec)
}

func (r *Runtime) ExportCampaign(ctx context.Context, campaignID uuid.UUID) (campaign.CampaignExportBundle, error) {
	if r == nil || r.effects == nil {
		return campaign.CampaignExportBundle{}, campaign.ErrServiceUnavailable()
	}
	return importexport.ExportCampaign(ctx, r.effects.CampaignImportExportHost(), campaignID)
}

func (r *Runtime) ImportCampaign(ctx context.Context, spec campaign.ImportCampaignSpec) (campaign.ImportCampaignResult, error) {
	if r == nil || r.effects == nil {
		return campaign.ImportCampaignResult{}, campaign.ErrServiceUnavailable()
	}
	return importexport.ImportCampaign(ctx, r.effects.CampaignImportExportHost(), spec)
}

func (r *Runtime) ImportMigrationCampaigns(ctx context.Context, spec campaign.ImportMigrationSpec) (campaign.ImportMigrationResult, error) {
	if r == nil || r.effects == nil {
		return campaign.ImportMigrationResult{}, campaign.ErrServiceUnavailable()
	}
	return r.effects.ImportMigrationCampaigns(ctx, spec)
}

func (r *Runtime) GetCampaignIntegrationHealth(ctx context.Context, campaignID uuid.UUID) (campaign.IntegrationHealthDTO, error) {
	if r == nil || r.effects == nil {
		return campaign.IntegrationHealthDTO{}, campaign.ErrServiceUnavailable()
	}
	return r.effects.GetCampaignIntegrationHealth(ctx, campaignID)
}

func (r *Runtime) CreateCampaign(ctx context.Context, spec campaign.CreateCampaignSpec) (uuid.UUID, error) {
	if r == nil || r.effects == nil {
		return uuid.Nil, campaign.ErrServiceUnavailable()
	}
	return createCampaign(ctx, r.PoolOrNil(), r.effects, spec)
}

func (r *Runtime) UpdateCampaignSchedule(ctx context.Context, campaignID uuid.UUID, startAt, endAt *time.Time, daypartHours []int16) error {
	if r == nil || r.effects == nil {
		return campaign.ErrServiceUnavailable()
	}
	return updateCampaignSchedule(ctx, r.PoolOrNil(), r.effects, campaignID, startAt, endAt, daypartHours)
}

func (r *Runtime) UpdateCampaignPacing(ctx context.Context, campaignID uuid.UUID, newMode string) (campaign.CampaignDTO, error) {
	if r == nil || r.effects == nil {
		return campaign.CampaignDTO{}, campaign.ErrServiceUnavailable()
	}
	return updateCampaignPacing(ctx, r.PoolOrNil(), r.effects, campaignID, newMode)
}

func (r *Runtime) GetCampaignStats(ctx context.Context, campaignID uuid.UUID, from, to time.Time, granularity string) (campaign.CampaignStatsDTO, error) {
	if r == nil {
		return campaign.CampaignStatsDTO{}, campaign.ErrServiceUnavailable()
	}
	return getCampaignStats(ctx, r.PoolOrNil(), r.clickhouseQuery, campaignID, from, to, granularity)
}

func (r *Runtime) PauseCampaign(ctx context.Context, campaignID uuid.UUID, reason string) error {
	if r == nil || r.effects == nil {
		return campaign.ErrServiceUnavailable()
	}
	return pauseCampaign(ctx, r.PoolOrNil(), r.effects, campaignID, reason)
}

func (r *Runtime) ResumeCampaign(ctx context.Context, campaignID uuid.UUID, reason string) error {
	if r == nil || r.effects == nil {
		return campaign.ErrServiceUnavailable()
	}
	return resumeCampaign(ctx, r.PoolOrNil(), r.effects, campaignID, reason, false)
}

func (r *Runtime) ResumeCampaignForPublish(ctx context.Context, campaignID uuid.UUID, force bool) error {
	if r == nil || r.effects == nil {
		return campaign.ErrServiceUnavailable()
	}
	return resumeCampaign(ctx, r.PoolOrNil(), r.effects, campaignID, "publish", force)
}

var _ campaign.CampaignReader = (*Runtime)(nil)
