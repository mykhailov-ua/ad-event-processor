package costsync

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// InsertManualCostLines upserts operator-entered spend rows into campaign_costs.
func InsertManualCostLines(ctx context.Context, pool *pgxpool.Pool, lines []CostLine) (int, error) {
	if pool == nil || len(lines) == 0 {
		return 0, nil
	}
	converter := &CurrencyConverter{pool: pool}
	fxCache, err := converter.PrepareFXCache(ctx, lines, lines[0].Date)
	if err != nil {
		return 0, err
	}
	usdAmounts := make([]int64, len(lines))
	for i, line := range lines {
		usdMicro, convErr := converter.ToUSDMicroCached(line.AmountMicro, line.Currency, fxCache)
		if convErr != nil {
			return 0, convErr
		}
		usdAmounts[i] = usdMicro
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	imported, err := insertCampaignCostsBatch(ctx, tx, lines, usdAmounts)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return imported, nil
}
