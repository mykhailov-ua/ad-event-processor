package controlplane

import (
	"context"
	"encoding/json"
	"time"

	"ad-event-processor/internal/campaign"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func campaignOwnerUserFilter(ctx context.Context) pgtype.UUID {
	return campaign.CampaignOwnerUserFilter(ctx)
}

func assertMediaBuyerCampaignAccess(ctx context.Context, camp db.Campaign) error {
	return campaign.AssertMediaBuyerCampaignAccess(ctx, camp)
}

func campaignRevision(updatedAt string) string {
	return campaign.CampaignRevision(updatedAt)
}

func resolveScheduleStatus(now time.Time, startAt, endAt *time.Time) db.CampaignStatusType {
	return campaign.ResolveScheduleStatus(now, startAt, endAt)
}

func validateDaypartHours(hours []int16) error {
	return campaign.ValidateDaypartHours(hours)
}

func validateSchedule(startAt, endAt *time.Time) error {
	return campaign.ValidateSchedule(startAt, endAt)
}

func countriesOrEmpty(c []string) []string {
	return campaign.CountriesOrEmpty(c)
}

func daypartOrEmpty(hours []int16) []int16 {
	return campaign.DaypartOrEmpty(hours)
}

func defaultTimezone(raw string) string {
	return campaign.DefaultTimezone(raw)
}

func ForecastRetryAfterSec() int {
	return campaign.ForecastRetryAfterSec()
}

func parseFlowPaths(raw json.RawMessage) ([]FlowPathDTO, error) {
	return campaign.ParseFlowPaths(raw)
}

func buildCampaignFlowValidateResponse(paths []FlowPathDTO) FlowValidateResponseDTO {
	return campaign.BuildCampaignFlowValidateResponse(paths)
}

func attachCampaignPresentation(ctx context.Context, dto *CampaignDTO) {
	campaign.AttachCampaignPresentation(ctx, dto)
}

func clickQueryParamsFromRaw(raw []byte) map[string]string {
	return campaign.ClickQueryParamsFromRaw(raw)
}

func nonNilUUID(id uuid.UUID) *uuid.UUID {
	return campaign.NonNilUUID(id)
}

func ApplyOnboardingTemplate(key string) (CampaignWizardStored, error) {
	return campaign.ApplyOnboardingTemplate(key)
}

var ErrClickHouseNotConfigured = campaign.ErrClickHouseNotConfigured
