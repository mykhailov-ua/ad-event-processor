package campaign

import (
	"net/http"
	"sort"
	"strings"
)

func filterAndSortCampaigns(items []CampaignDTO, q, sortField, order, pacingMode string) []CampaignDTO {
	filtered := make([]CampaignDTO, 0, len(items))
	q = strings.ToLower(strings.TrimSpace(q))
	for _, item := range items {
		if q != "" && !strings.Contains(strings.ToLower(item.Name), q) && !strings.Contains(strings.ToLower(item.ID), q) {
			continue
		}
		if pacingMode != "" && !strings.EqualFold(item.PacingMode, pacingMode) {
			continue
		}
		filtered = append(filtered, item)
	}
	if sortField == "" {
		sortField = "updated_at"
	}
	sort.Slice(filtered, func(i, j int) bool {
		less := false
		switch sortField {
		case "name":
			less = strings.ToLower(filtered[i].Name) < strings.ToLower(filtered[j].Name)
		case "spend":
			less = filtered[i].CurrentSpend < filtered[j].CurrentSpend
		case "budget_limit":
			less = filtered[i].BudgetLimit < filtered[j].BudgetLimit
		default:
			less = filtered[i].UpdatedAt < filtered[j].UpdatedAt
		}
		if order == "asc" {
			return less
		}
		return !less
	})
	return filtered
}

func CampaignRevision(updatedAt string) string {
	return campaignRevision(updatedAt)
}

func campaignRevision(updatedAt string) string {
	return strings.TrimSpace(updatedAt)
}

func resolveExpectedRevision(r *http.Request, req *PatchCampaignRequest) {
	if req.ExpectedRevision != nil && strings.TrimSpace(*req.ExpectedRevision) != "" {
		return
	}
	ifMatch := strings.TrimSpace(r.Header.Get("If-Match"))
	if ifMatch != "" {
		req.ExpectedRevision = &ifMatch
	}
}
