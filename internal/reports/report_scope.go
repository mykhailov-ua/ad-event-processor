package reports

import (
	"context"
	"net/http"
	"strings"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func ParseOptionalCampaignGroupFilter(r *http.Request) (uuid.UUID, bool, error) {
	raw := strings.TrimSpace(r.URL.Query().Get("group_id"))
	if raw == "" {
		return uuid.Nil, false, nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, false, err
	}
	return id, true, nil
}

func ResolveReportCampaignIDs(
	ctx context.Context,
	pool *pgxpool.Pool,
	customerID uuid.UUID,
	campaignFilter uuid.UUID,
	campaignFilterSet bool,
	groupFilter uuid.UUID,
	groupFilterSet bool,
) ([]uuid.UUID, error) {
	campaignIDs, err := listCustomerCampaignIDs(ctx, pool, customerID)
	if err != nil {
		return nil, err
	}
	if groupFilterSet {
		groupIDs, err := listCampaignIDsInGroup(ctx, pool, customerID, groupFilter)
		if err != nil {
			return nil, err
		}
		campaignIDs = intersectCampaignIDs(campaignIDs, groupIDs)
	}
	return narrowCampaignIDs(campaignIDs, campaignFilter, campaignFilterSet), nil
}

func listCampaignIDsInGroup(ctx context.Context, pool *pgxpool.Pool, customerID, groupID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := db.New(pool).ListCampaignIDsByGroup(ctx, db.ListCampaignIDsByGroupParams{
		CustomerID:      domain.ToUUID(customerID),
		CampaignGroupID: domain.ToUUID(groupID),
	})
	if err != nil {
		return nil, err
	}
	out := make([]uuid.UUID, 0, len(rows))
	for i := range rows {
		if !rows[i].Valid {
			continue
		}
		out = append(out, uuid.UUID(rows[i].Bytes))
	}
	return out, nil
}

func intersectCampaignIDs(scope []uuid.UUID, filter []uuid.UUID) []uuid.UUID {
	if len(filter) == 0 {
		return scope
	}
	allowed := make(map[uuid.UUID]struct{}, len(filter))
	for _, id := range filter {
		allowed[id] = struct{}{}
	}
	out := make([]uuid.UUID, 0, len(scope))
	for _, id := range scope {
		if _, ok := allowed[id]; ok {
			out = append(out, id)
		}
	}
	return out
}
