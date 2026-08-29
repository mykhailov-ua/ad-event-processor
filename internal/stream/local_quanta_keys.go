package stream

import (
	"hash/crc32"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/filter"

	"github.com/google/uuid"
)

const hexChars = "0123456789abcdef"

func monotonicNano() int64 {
	return filter.MonotonicNano()
}

func spreadHighVolumeShard(shardCount int, campaignID uuid.UUID, subSlot int) int {
	if shardCount <= 1 {
		return 0
	}
	h := domain.CRC32Castagnoli(&campaignID) + uint32(subSlot)
	return int(h % uint32(shardCount))
}

func fcapKeyPrefixForDebit(camp *domain.Campaign, userID, clickID string) string {
	if camp == nil {
		return ""
	}
	sub := debitSubSlot(camp, userID, clickID)
	return domain.FcapKeyPrefixSub(camp.ID, camp.BrandFcapKey, sub)
}

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
	h := computeCompositeHashUUID(camp.ID, key)
	return int(h % uint32(n))
}

func budgetQuotaKeyForDebit(campaignID uuid.UUID, subSlot int) string {
	if subSlot <= 0 {
		return domain.BudgetQuotaKey(campaignID)
	}
	return domain.BudgetQuotaKeySub(campaignID, subSlot)
}

func computeCompositeHashUUID(campaignID uuid.UUID, userID []byte) uint32 {
	var crc uint32
	var started bool

	if campaignID != uuid.Nil {
		var buf [36]byte
		formatUUIDCanonical(&buf, campaignID)
		crc = crc32IEEEInplace36(&buf)
		started = true
	}
	if len(userID) > 0 {
		if started {
			crc = crc32.Update(crc, crc32.IEEETable, userID)
		} else {
			crc = crc32.ChecksumIEEE(userID)
		}
	}
	if !started && len(userID) == 0 {
		return 0
	}
	return crc
}

func crc32IEEEInplace36(b *[36]byte) uint32 {
	crc := ^uint32(0)
	tab := crc32.IEEETable
	for i := range 36 {
		crc = tab[byte(crc)^b[i]] ^ crc>>8
	}
	return ^crc
}

func formatUUIDCanonical(dst *[36]byte, id uuid.UUID) {
	b := dst[:]
	b[0] = hexChars[id[0]>>4]
	b[1] = hexChars[id[0]&0xf]
	b[2] = hexChars[id[1]>>4]
	b[3] = hexChars[id[1]&0xf]
	b[4] = hexChars[id[2]>>4]
	b[5] = hexChars[id[2]&0xf]
	b[6] = hexChars[id[3]>>4]
	b[7] = hexChars[id[3]&0xf]
	b[8] = '-'
	b[9] = hexChars[id[4]>>4]
	b[10] = hexChars[id[4]&0xf]
	b[11] = hexChars[id[5]>>4]
	b[12] = hexChars[id[5]&0xf]
	b[13] = '-'
	b[14] = hexChars[id[6]>>4]
	b[15] = hexChars[id[6]&0xf]
	b[16] = hexChars[id[7]>>4]
	b[17] = hexChars[id[7]&0xf]
	b[18] = '-'
	b[19] = hexChars[id[8]>>4]
	b[20] = hexChars[id[8]&0xf]
	b[21] = hexChars[id[9]>>4]
	b[22] = hexChars[id[9]&0xf]
	b[23] = '-'
	b[24] = hexChars[id[10]>>4]
	b[25] = hexChars[id[10]&0xf]
	b[26] = hexChars[id[11]>>4]
	b[27] = hexChars[id[11]&0xf]
	b[28] = hexChars[id[12]>>4]
	b[29] = hexChars[id[12]&0xf]
	b[30] = hexChars[id[13]>>4]
	b[31] = hexChars[id[13]&0xf]
	b[32] = hexChars[id[14]>>4]
	b[33] = hexChars[id[14]&0xf]
	b[34] = hexChars[id[15]>>4]
	b[35] = hexChars[id[15]&0xf]
}
