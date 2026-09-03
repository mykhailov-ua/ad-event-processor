package campaign

import (
	"context"

	db "ad-event-processor/internal/domain/db"
)

func ListCampaignsUsesStatsJoin(filter ListCampaignsFilter) bool {
	return IsCampaignListStatsSortField(filter.SortField) && filter.StatsRangeSet
}

func QueryListCampaignRows(ctx context.Context, q *db.Queries, filter ListCampaignsFilter) ([]db.Campaign, error) {
	if ListCampaignsUsesStatsJoin(filter) {
		return q.ListCampaignsSortedByStats(ctx, CampaignListSortedByStatsParamsFromFilter(filter))
	}
	return q.ListCampaigns(ctx, CampaignListParamsFromFilter(filter))
}
