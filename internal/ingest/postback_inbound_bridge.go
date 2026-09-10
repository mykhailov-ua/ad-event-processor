package ingest

import (
	"time"

	"ad-event-processor/internal/postback/inbound"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPostbackInboundStore(pool *pgxpool.Pool) *inbound.Store {
	return inbound.NewStore(pool, 60*time.Second)
}
