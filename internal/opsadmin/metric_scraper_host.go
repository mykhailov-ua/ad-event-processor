package opsadmin

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type OpsMetricScraperHost interface {
	GetPool() *pgxpool.Pool
	WithPostgresLow(ctx context.Context, fn func(context.Context) error) error
	StartBackgroundWorker(fn func())
}
