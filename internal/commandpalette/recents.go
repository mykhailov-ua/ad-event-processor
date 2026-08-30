package commandpalette

import (
	"context"
	"fmt"
	"strings"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const maxRecentsPerUser = 20

type recentsQuerier interface {
	ListCommandPaletteRecents(ctx context.Context, arg db.ListCommandPaletteRecentsParams) ([]db.ListCommandPaletteRecentsRow, error)
	UpsertCommandPaletteRecent(ctx context.Context, arg db.UpsertCommandPaletteRecentParams) error
	PruneCommandPaletteRecents(ctx context.Context, arg db.PruneCommandPaletteRecentsParams) error
}

type RecentsStore struct {
	q recentsQuerier
}

func NewRecentsStore(pool *pgxpool.Pool) *RecentsStore {
	if pool == nil {
		return nil
	}
	return &RecentsStore{q: db.New(pool)}
}

func (st *RecentsStore) ListRecents(ctx context.Context, customerID, userID uuid.UUID) ([]ItemDTO, error) {
	if st == nil || st.q == nil {
		return nil, fmt.Errorf("command palette recents store unavailable")
	}
	if customerID == uuid.Nil {
		return nil, fmt.Errorf("customer_id is required")
	}
	if userID == uuid.Nil {
		return nil, fmt.Errorf("user_id is required")
	}
	rows, err := st.q.ListCommandPaletteRecents(ctx, db.ListCommandPaletteRecentsParams{
		CustomerID: domain.ToUUID(customerID),
		UserID:     domain.ToUUID(userID),
	})
	if err != nil {
		return nil, err
	}
	out := make([]ItemDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, recentRowToItem(row))
	}
	return out, nil
}

func (st *RecentsStore) RecordRecent(ctx context.Context, customerID, userID uuid.UUID, item ItemDTO) error {
	if st == nil || st.q == nil {
		return fmt.Errorf("command palette recents store unavailable")
	}
	if customerID == uuid.Nil {
		return fmt.Errorf("customer_id is required")
	}
	if userID == uuid.Nil {
		return fmt.Errorf("user_id is required")
	}
	itemID := strings.TrimSpace(item.ID)
	kind := strings.TrimSpace(item.Kind)
	label := strings.TrimSpace(item.Label)
	href := strings.TrimSpace(item.Href)
	if itemID == "" || kind == "" || label == "" || href == "" {
		return fmt.Errorf("item id, kind, label, and href are required")
	}
	err := st.q.UpsertCommandPaletteRecent(ctx, db.UpsertCommandPaletteRecentParams{
		CustomerID: domain.ToUUID(customerID),
		UserID:     domain.ToUUID(userID),
		ItemID:     itemID,
		Kind:       kind,
		Label:      label,
		Href:       href,
		Meta:       optionalText(item.Meta),
		GroupName:  optionalText(item.Group),
	})
	if err != nil {
		return err
	}
	return st.q.PruneCommandPaletteRecents(ctx, db.PruneCommandPaletteRecentsParams{
		CustomerID: domain.ToUUID(customerID),
		UserID:     domain.ToUUID(userID),
		KeepMax:    int32(maxRecentsPerUser),
	})
}

func recentRowToItem(row db.ListCommandPaletteRecentsRow) ItemDTO {
	item := ItemDTO{
		ID:    row.ItemID,
		Kind:  row.Kind,
		Label: row.Label,
		Href:  row.Href,
	}
	if row.Meta.Valid {
		item.Meta = row.Meta.String
	}
	if row.Group.Valid {
		item.Group = row.Group.String
	}
	return item
}

func optionalText(value string) pgtype.Text {
	value = strings.TrimSpace(value)
	if value == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: value, Valid: true}
}
