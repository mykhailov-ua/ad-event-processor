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

const (
	MigrationFenceKeyPrefix = domain.MigrationFenceKeyPrefix
	BudgetFrozenKeyPrefix   = domain.BudgetFrozenKeyPrefix
)

func appendCampaignHashTag(dst []byte, id uuid.UUID) []byte {
	return domain.AppendCampaignHashTag(dst, id)
}

func appendCampaignSubHashTag(dst []byte, id uuid.UUID, sub int) []byte {
	return domain.AppendCampaignSubHashTag(dst, id, sub)
}

func budgetQuotaKeySub(id uuid.UUID, sub int) string {
	return domain.BudgetQuotaKeySub(id, sub)
}

func fcapKeyPrefixSub(campaignID uuid.UUID, brandFcapKey string, sub int) string {
	return domain.FcapKeyPrefixSub(campaignID, brandFcapKey, sub)
}

func campaignHashTag(id uuid.UUID) string {
	return domain.CampaignHashTag(id)
}

func crc32Castagnoli(data *uuid.UUID) uint32 {
	return domain.CRC32Castagnoli(data)
}
