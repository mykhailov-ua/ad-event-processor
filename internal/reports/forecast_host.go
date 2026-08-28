package reports

import (
	"ad-event-processor/internal/database"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ForecastHost interface {
	ForecastPool() *pgxpool.Pool
	ForecastClickHouseQuery() *database.ClickHouseQuery
}
