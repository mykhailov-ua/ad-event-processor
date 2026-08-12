package ingestion

import (
	"context"
	db "github.com/bidshard/ad-event-processor/internal/domain/db"
)

func LoadSupplyChainAllowlist(ctx context.Context, q *db.Queries) (*SupplyChainAllowlistSnapshot, error) {
	if q == nil {
		return &SupplyChainAllowlistSnapshot{Allowed: make(map[string]struct{})}, nil
	}
	rows, err := q.ListSellers(ctx)
	if err != nil {
		return nil, err
	}
	domains := make([]string, 0, len(rows))
	sellerIDs := make([]string, 0, len(rows))
	for _, row := range rows {
		domains = append(domains, row.Domain)
		sellerIDs = append(sellerIDs, row.SellerID)
	}
	return BuildSupplyChainAllowlistFromSellers(domains, sellerIDs), nil
}
