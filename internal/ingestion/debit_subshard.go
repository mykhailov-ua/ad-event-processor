package ingestion

import (
	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
)

func debitSubSlot(camp *domain.Campaign, userID, clickID string) int {
	n := camp.DebitSubShardCount()
	if n <= 1 {
		return 0
	}
	var key []byte
	if userID != "" {
		key = UnsafeBytes(userID)
	} else if clickID != "" {
		key = UnsafeBytes(clickID)
	}
	if len(key) == 0 {
		return 0
	}
	h := ComputeCompositeHashUUID(camp.ID, key)
	return int(h % uint32(n))
}

func spreadHighVolumeShard(shardCount int, campaignID uuid.UUID, subSlot int) int {
	if shardCount <= 1 {
		return 0
	}
	h := domain.CRC32Castagnoli(&campaignID) + uint32(subSlot)
	return int(h % uint32(shardCount))
}

func appendBudgetQuotaKey(dst []byte, campaignID uuid.UUID, subSlot int) []byte {
	if subSlot <= 0 {
		dst = appendCampaignHashTag(dst, campaignID)
	} else {
		dst = domain.AppendCampaignSubHashTag(dst, campaignID, subSlot)
	}
	dst = append(dst, "budget:quota:"...)
	return appendUUID(dst, campaignID)
}

func budgetQuotaKeyForDebit(campaignID uuid.UUID, subSlot int) string {
	if subSlot <= 0 {
		return budgetQuotaKey(campaignID)
	}
	return domain.BudgetQuotaKeySub(campaignID, subSlot)
}

func fcapKeyPrefixForDebit(camp *domain.Campaign, userID, clickID string) string {
	if camp == nil {
		return ""
	}
	sub := debitSubSlot(camp, userID, clickID)
	return domain.FcapKeyPrefixSub(camp.ID, camp.BrandFcapKey, sub)
}
