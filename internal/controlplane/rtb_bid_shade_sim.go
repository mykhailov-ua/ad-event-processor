package controlplane

import (
	"context"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

type RtbBidShadeSimulator func(ctx context.Context, pool *pgxpool.Pool, cfg *config.Config, in domain.RtbBidShadeInput) (domain.RtbBidShadeOutput, error)
