package fraudadmin

import (
	"context"

	"ad-event-processor/internal/database"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ModeratorCorpusHost interface {
	ModeratorCorpusPool() *pgxpool.Pool
	ModeratorCorpusClickHouse() *database.ClickHouseQuery
	RefreshModeratorCorpusFeed(ctx context.Context) error
	ModeratorCorpusFeedLastRefresh(ctx context.Context) (string, bool)
}

type ModeratorCorpus struct {
	host ModeratorCorpusHost
}

func NewModeratorCorpus(host ModeratorCorpusHost) *ModeratorCorpus {
	return &ModeratorCorpus{host: host}
}
