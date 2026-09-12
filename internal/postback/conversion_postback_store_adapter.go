package postback

import (
	"context"

	db "ad-event-processor/internal/domain/db"

	"github.com/jackc/pgx/v5/pgtype"
)

type conversionPostbackQueries struct {
	inner *db.Queries
}

func (c *conversionPostbackQueries) ListPostbackConfigsByCampaignIDs(ctx context.Context, ids []pgtype.UUID) ([]db.PostbackConfig, error) {
	return c.inner.ListPostbackConfigsByCampaignIDs(ctx, ids)
}

func (c *conversionPostbackQueries) ListOutboundPostbacksByCampaignIDs(ctx context.Context, ids []pgtype.UUID) ([]db.CampaignOutboundPostback, error) {
	rows, err := c.inner.ListOutboundPostbacksByCampaignIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]db.CampaignOutboundPostback, 0, len(rows))
	for i := range rows {
		out = append(out, db.CampaignOutboundPostbackFromListIDs(rows[i]))
	}
	return out, nil
}

func (c *conversionPostbackQueries) ListCampaignsByIDs(ctx context.Context, ids []pgtype.UUID) ([]db.Campaign, error) {
	return c.inner.ListCampaignsByIDs(ctx, ids)
}

func (c *conversionPostbackQueries) CreatePostbackOutboxEventsBatch(ctx context.Context, arg db.CreatePostbackOutboxEventsBatchParams) error {
	return c.inner.CreatePostbackOutboxEventsBatch(ctx, arg)
}
