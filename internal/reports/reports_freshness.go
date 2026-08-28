package reports

import (
	"context"
	"time"

	"ad-event-processor/internal/database"
)

func DataFreshnessFromClickHouse(ctx context.Context, clickhouseQuery *database.ClickHouseQuery) DataFreshnessDTO {
	dto := DataFreshnessDTO{
		AsOf:        time.Now().UTC().Format(time.RFC3339),
		Consistency: "eventual",
	}
	if clickhouseQuery == nil {
		dto.Stale = true
		return dto
	}
	lag, err := clickhouseQuery.IngestionLag(ctx)
	if err != nil {
		dto.Stale = true
		return dto
	}
	dto.Stale, dto.CHLagSeconds = database.Freshness(lag, 5*time.Minute)
	return dto
}
