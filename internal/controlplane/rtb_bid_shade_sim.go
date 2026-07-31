package controlplane

import (
	"context"

	"espx/internal/config"
	"espx/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

type RtbBidShadeSimulator func(ctx context.Context, pool *pgxpool.Pool, cfg *config.Config, in domain.RtbBidShadeInput) (domain.RtbBidShadeOutput, error)
