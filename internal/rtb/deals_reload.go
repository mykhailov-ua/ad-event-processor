package rtb

import (
	"context"
	"encoding/binary"

	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
)

type DealCatalog interface {
	UpdateDeals(deals []DealData)
}

func ReloadDeals(ctx context.Context, q *db.Queries, catalog DealCatalog) error {
	if catalog == nil {
		return nil
	}
	rows, err := q.ListRtbDeals(ctx)
	if err != nil {
		return err
	}
	deals := make([]DealData, 0, len(rows))
	for _, row := range rows {
		deals = append(deals, dealRowToData(row))
	}
	catalog.UpdateDeals(deals)
	return nil
}

func dealRowToData(row db.RtbDeal) DealData {
	id := uuid.UUID(row.CustomerID.Bytes)
	return DealData{
		DealID:     row.DealID,
		FloorMicro: row.FloorMicro,
		GeoMask:    uint64(row.GeoMask),
		CatMask:    uint64(row.CatMask),
		PacingOpen: DealPacingOpen(row.Pacing),
		Seats:      row.Seats,
		CustomerID: CustomerID(binary.BigEndian.Uint64(id[:8])),
	}
}
