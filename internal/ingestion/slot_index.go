package ingestion

import "github.com/google/uuid"

func CampaignSlotIndex(id uuid.UUID) int16 {
	return int16(crc32Castagnoli(&id) & SlotMask)
}

func FilterCampaignIDsBySlot(ids []uuid.UUID, slot int16) []uuid.UUID {
	out := make([]uuid.UUID, 0, len(ids)/4)
	for _, id := range ids {
		if CampaignSlotIndex(id) == slot {
			out = append(out, id)
		}
	}
	return out
}
