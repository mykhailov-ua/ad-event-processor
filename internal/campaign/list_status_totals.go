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

func ApplyCampaignStatusCount(totals *CampaignStatusTotalsDTO, status string, count int64) {
	if totals == nil {
		return
	}
	totals.Total += count
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "ACTIVE":
		totals.Active += count
	case "PAUSED":
		totals.Paused += count
	case "ARCHIVED":
		totals.Archived += count
	}
}
