package fraudadmin

import (
	"ad-event-processor/internal/database"

	"github.com/jackc/pgx/v5/pgxpool"
)

type MLShadowDeltaSnapshotHost interface {
	SnapshotPool() *pgxpool.Pool
	ClickHouseQuery() *database.ClickHouseQuery
}
