package domain

import (
	"context"
	"encoding/binary"

	db "espx/internal/domain/db"
	"espx/internal/rtb"

	"github.com/google/uuid"
)

type RtbDealCatalog interface {
	UpdateDeals(deals []rtb.DealData)
}

func ReloadRtbDeals(ctx context.Context, q *db.Queries, catalog RtbDealCatalog) error {
	if catalog == nil {
		return nil
	}
	rows, err := q.ListRtbDeals(ctx)
	if err != nil {
		return err
	}
	deals := make([]rtb.DealData, 0, len(rows))
	for _, row := range rows {
		deals = append(deals, rtbDealRowToData(row))
	}
	catalog.UpdateDeals(deals)
	return nil
}

func rtbDealRowToData(row db.RtbDeal) rtb.DealData {
	id := uuid.UUID(row.CustomerID.Bytes)
	return rtb.DealData{
		DealID:     row.DealID,
		FloorMicro: row.FloorMicro,
		GeoMask:    uint64(row.GeoMask),
		CatMask:    uint64(row.CatMask),
		PacingOpen: rtb.DealPacingOpen(row.Pacing),
		Seats:      row.Seats,
		CustomerID: rtb.CustomerID(binary.BigEndian.Uint64(id[:8])),
	}
}
