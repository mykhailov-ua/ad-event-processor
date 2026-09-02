package dashboardadmin

import (
	"context"
	"fmt"
	"time"

	"ad-event-processor/internal/reports"

	"github.com/google/uuid"
)

func (p *Portfolio) GetBuyerDrilldown(
	ctx context.Context,
	customerID uuid.UUID,
	campaignID uuid.UUID,
	from, to time.Time,
	filter reports.DashboardDrilldownFilter,
) (reports.DashboardBreakdownTableDTO, error) {
	if p == nil || p.host == nil {
		return reports.DashboardBreakdownTableDTO{}, fmt.Errorf("portfolio service unavailable")
	}
	if customerID == uuid.Nil {
		return reports.DashboardBreakdownTableDTO{}, p.host.ErrValidation("customer_id is required")
	}
	if campaignID == uuid.Nil {
		return reports.DashboardBreakdownTableDTO{}, p.host.ErrValidation("campaign_id is required")
	}
	if err := reports.ValidateChartRange(from, to); err != nil {
		return reports.DashboardBreakdownTableDTO{}, err
	}

	campaigns, err := p.listAllCampaigns(ctx, customerID, "")
	if err != nil {
		return reports.DashboardBreakdownTableDTO{}, err
	}
	allowed := false
	for _, campaign := range campaigns {
		if campaign.ID == campaignID.String() {
			allowed = true
			break
		}
	}
	if !allowed {
		return reports.DashboardBreakdownTableDTO{}, p.host.ErrValidation("campaign not found for customer")
	}

	chQuery := p.host.ClickHouseQuery()
	if chQuery == nil {
		return reports.DashboardBreakdownTableDTO{Rows: []reports.DashboardBreakdownRowDTO{}}, nil
	}
	clickhouseCtx, cancel := context.WithTimeout(ctx, p.host.ReportCHTimeout())
	defer cancel()
	return reports.QueryCampaignDrilldownCH(
		clickhouseCtx,
		chQuery,
		[]uuid.UUID{campaignID},
		from,
		to,
		filter,
		buyerDrilldownTopN,
	)
}

const buyerDrilldownTopN = 25
