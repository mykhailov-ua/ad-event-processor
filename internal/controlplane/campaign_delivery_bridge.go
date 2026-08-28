package controlplane

import (
	"context"
	"strings"
	"time"

	"ad-event-processor/internal/automation"
	"ad-event-processor/internal/campaign"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/shardadmin"

	"github.com/google/uuid"
)

var (
	_ campaign.DeliveryHost        = (*Service)(nil)
	_ campaign.Effects             = (*Service)(nil)
	_ campaign.LoopHost            = (*Service)(nil)
	_ campaign.VPPHost             = (*Service)(nil)
	_ campaign.DrainHost           = (*Service)(nil)
	_ campaign.WizardHost          = (*Service)(nil)
	_ campaign.TemplateCatalogHost = (*Service)(nil)
	_ automation.LicenseGate       = (*Service)(nil)
)

var ErrClickHouseNotConfigured = campaign.ErrClickHouseNotConfigured

const campaignExportVersion = campaign.CampaignExportVersion

var errCampaignWizardIncomplete = campaign.ErrCampaignWizardIncomplete

const (
	wizardStepTrafficSource       = campaign.WizardStepTrafficSource
	wizardStepIntegrationTemplate = campaign.WizardStepIntegrationTemplate
	wizardStepFlowSkeleton        = campaign.WizardStepFlowSkeleton
	wizardStepBudget              = campaign.WizardStepBudget
	wizardStepReview              = campaign.WizardStepReview
)

func assertMediaBuyerCampaignAccess(ctx context.Context, camp db.Campaign) error {
	return campaign.AssertMediaBuyerCampaignAccess(ctx, camp)
}

func resolveScheduleStatus(now time.Time, startAt, endAt *time.Time) db.CampaignStatusType {
	return campaign.ResolveScheduleStatus(now, startAt, endAt)
}

func validateDaypartHours(hours []int16) error {
	return campaign.ValidateDaypartHours(hours)
}

func ForecastRetryAfterSec() int {
	return campaign.ForecastRetryAfterSec()
}

func (s *Service) cloneCampaignPlacementBlocks(ctx context.Context, sourceID, destID uuid.UUID) error {
	if s == nil || len(s.redisShards) == 0 {
		return nil
	}
	redisClient := s.redisClientForCampaign(sourceID)
	if redisClient == nil {
		return nil
	}
	key := domain.PlacementBlacklistKey(sourceID)
	placements, err := redisClient.HKeys(ctx, key).Result()
	if err != nil {
		return err
	}
	if len(placements) == 0 {
		return nil
	}
	destKey := domain.PlacementBlacklistKey(destID)
	for _, placementID := range placements {
		placementID = strings.TrimSpace(placementID)
		if placementID == "" {
			continue
		}
		if err := shardadmin.SyncGlobalHashFieldToAllShards(ctx, s.redisShards, destKey, placementID, "1", false); err != nil {
			return err
		}
	}
	return nil
}
