package campaign

import (
	"strings"
)

type CampaignStatusTotalsDTO struct {
	Active   int64 `json:"active"`
	Paused   int64 `json:"paused"`
	Archived int64 `json:"archived"`
	Total    int64 `json:"total"`
}

// FilterCampaignsByQuery applies name/id substring and pacing_mode filters in memory after SQL list.
// Used when status_totals cannot use SQL-only counts (see countCampaignStatusTotals probe path).
func FilterCampaignsByQuery(items []CampaignDTO, searchQuery, pacingMode string) []CampaignDTO {
	searchQuery = strings.ToLower(strings.TrimSpace(searchQuery))
	pacingMode = strings.TrimSpace(pacingMode)
	if searchQuery == "" && pacingMode == "" {
		return items
	}

	filtered := make([]CampaignDTO, 0, len(items))
	for _, item := range items {
		if searchQuery != "" && !strings.Contains(strings.ToLower(item.Name), searchQuery) && !strings.Contains(strings.ToLower(item.ID), searchQuery) {
			continue
		}
		if pacingMode != "" && !strings.EqualFold(item.PacingMode, pacingMode) {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func CountStatusTotalsFromItems(items []CampaignDTO) CampaignStatusTotalsDTO {
	var totals CampaignStatusTotalsDTO
	for _, item := range items {
		switch strings.ToUpper(strings.TrimSpace(item.Status)) {
		case "ACTIVE":
			totals.Active++
		case "PAUSED":
			totals.Paused++
		case "ARCHIVED":
			totals.Archived++
		}
		totals.Total++
	}
	return totals
}
