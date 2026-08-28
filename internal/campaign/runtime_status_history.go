package campaign

import (
	"context"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/pkg/coldpath"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func listStatusHistory(
	ctx context.Context,
	pool *pgxpool.Pool,
	campaignID uuid.UUID,
	limit, offset int32,
) ([]StatusHistoryDTO, int64, error) {
	if pool == nil {
		return nil, 0, errServiceUnavailable()
	}
	q := db.New(pool)
	cid := domain.ToUUID(campaignID)
	listParams := db.ListStatusHistoryParams{
		CampaignID: cid,
		Limit:      limit,
		Offset:     offset,
	}
	return coldpath.PaginatedList(
		func() (int64, error) { return q.CountStatusHistory(ctx, cid) },
		func() ([]db.CampaignStatusHistory, error) { return q.ListStatusHistory(ctx, listParams) },
		statusHistoryToDTO,
	)
}
