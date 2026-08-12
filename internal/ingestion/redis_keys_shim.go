package ingestion

import (
	"github.com/bidshard/ad-event-processor/internal/domain"

	"github.com/google/uuid"
)

func BudgetCampaignKey(id uuid.UUID) string {
	return domain.BudgetCampaignKey(id)
}

func CampaignSyncKey(id uuid.UUID) string {
	return domain.CampaignSyncKey(id)
}

func PlacementBlacklistKey(campaignID uuid.UUID) string {
	return domain.PlacementBlacklistKey(campaignID)
}

func RedisClusterSlot(key string) int {
	return domain.RedisClusterSlot(key)
}
