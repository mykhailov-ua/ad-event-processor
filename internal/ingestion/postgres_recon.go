package ingestion

import (
	"context"
	"fmt"

	"github.com/bidshard/ad-event-processor/internal/domain"
	db "github.com/bidshard/ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SQLPostgresConn struct {
	queries *db.Queries
}

func NewSQLPostgresConn(pool *pgxpool.Pool) *SQLPostgresConn {
	if pool == nil {
		return nil
	}
	return &SQLPostgresConn{queries: db.New(pool)}
}

func (p *SQLPostgresConn) GetCampaignSpend(ctx context.Context, campaignID uuid.UUID) (int64, error) {
	spends, err := p.GetCampaignSpends(ctx, []uuid.UUID{campaignID})
	if err != nil {
		return 0, err
	}
	spend, ok := spends[campaignID]
	if !ok {
		return 0, nil
	}
	return spend, nil
}

func (p *SQLPostgresConn) GetCampaignSpends(ctx context.Context, campaignIDs []uuid.UUID) (map[uuid.UUID]int64, error) {
	out := make(map[uuid.UUID]int64, len(campaignIDs))
	if p == nil || p.queries == nil || len(campaignIDs) == 0 {
		return out, nil
	}
	pgIDs := make([]pgtype.UUID, len(campaignIDs))
	for i, id := range campaignIDs {
		pgIDs[i] = domain.ToUUID(id)
	}
	rows, err := p.queries.ListCampaignSpendsByIDs(ctx, pgIDs)
	if err != nil {
		return nil, fmt.Errorf("list campaign spends: %w", err)
	}
	for _, row := range rows {
		id, err := uuid.FromBytes(row.ID.Bytes[:])
		if err != nil {
			continue
		}
		out[id] = row.CurrentSpend
	}
	return out, nil
}

func (p *SQLPostgresConn) UpdateCampaignSpend(ctx context.Context, campaignID uuid.UUID, currentSpend int64) error {
	if p == nil || p.queries == nil {
		return fmt.Errorf("postgres conn unavailable")
	}
	return p.queries.UpdateCampaignSpend(ctx, db.UpdateCampaignSpendParams{
		ID:           domain.ToUUID(campaignID),
		CurrentSpend: currentSpend,
	})
}

func (p *SQLPostgresConn) GetCampaignBudgetLimit(ctx context.Context, campaignID uuid.UUID) (int64, error) {
	if p == nil || p.queries == nil {
		return 0, fmt.Errorf("postgres conn unavailable")
	}
	row, err := p.queries.GetCampaignBudget(ctx, domain.ToUUID(campaignID))
	if err != nil {
		return 0, err
	}
	return row.BudgetLimit, nil
}

func (p *SQLPostgresConn) MarkEventIdempotent(ctx context.Context, clickID string) (bool, error) {
	return false, fmt.Errorf("mark event idempotent not implemented on SQLPostgresConn")
}
