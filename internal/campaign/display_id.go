package campaign

import (
	"fmt"

	"github.com/google/uuid"
)

// CampaignDisplayID maps a campaign UUID to an eight-digit id in [10000000, 99999999].
// Polynomial hash (base 31) must match web/src/domains/campaigns/list/campaign_display_id.ts.
func CampaignDisplayID(id uuid.UUID) string {
	var hash uint64
	for _, b := range id {
		hash = hash*31 + uint64(b)
	}
	return fmt.Sprintf("%08d", 10000000+(hash%90000000))
}

// CampaignDisplayIDFromString parses id as UUID when possible; otherwise hashes the raw string.
func CampaignDisplayIDFromString(id string) string {
	parsed, err := uuid.Parse(id)
	if err != nil {
		var hash uint64
		for i := 0; i < len(id); i++ {
			hash = hash*31 + uint64(id[i])
		}
		return fmt.Sprintf("%08d", 10000000+(hash%90000000))
	}
	return CampaignDisplayID(parsed)
}

func attachCampaignDisplayID(dto *CampaignDTO) {
	if dto == nil || dto.ID == "" {
		return
	}
	dto.DisplayID = CampaignDisplayIDFromString(dto.ID)
}
