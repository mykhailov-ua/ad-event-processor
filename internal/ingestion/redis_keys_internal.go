package ingestion

import (
	"github.com/bidshard/ad-event-processor/internal/domain"

	"github.com/google/uuid"
)

func budgetCampaignKey(id uuid.UUID) string {
	return domain.BudgetCampaignKey(id)
}

func campaignSyncKey(id uuid.UUID) string {
	return domain.CampaignSyncKey(id)
}

func customerSyncKey(campaignID, customerID uuid.UUID) string {
	return domain.CustomerSyncKey(campaignID, customerID)
}

func budgetQuotaKey(id uuid.UUID) string {
	return domain.BudgetQuotaKey(id)
}

func fcapKeyPrefix(campaignID uuid.UUID, brandFcapKey string) string {
	return domain.FcapKeyPrefix(campaignID, brandFcapKey)
}

func dailySpendKeyPrefix(campaignID uuid.UUID) string {
	return domain.DailySpendKeyPrefix(campaignID)
}

const MigrationFenceKeyPrefix = domain.MigrationFenceKeyPrefix
const BudgetFrozenKeyPrefix = domain.BudgetFrozenKeyPrefix

func appendCampaignHashTag(dst []byte, id uuid.UUID) []byte {
	return domain.AppendCampaignHashTag(dst, id)
}

func campaignHashTag(id uuid.UUID) string {
	return domain.CampaignHashTag(id)
}

func crc32Castagnoli(data *uuid.UUID) uint32 {
	return domain.CRC32Castagnoli(data)
}
