package smartalerts

import (
	"context"
	"time"

	"ad-event-processor/internal/database"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Host interface {
	Pool() *pgxpool.Pool
	ClickHouseQuery() *database.ClickHouseQuery
	DrainStuckThresholdSec() int
	AlertDrainStuck(ctx context.Context, version int32, slot int16, state, lastError string, updatedAt time.Time)
}

type Store struct {
	host Host
}

func NewStore(host Host) *Store {
	return &Store{host: host}
}
