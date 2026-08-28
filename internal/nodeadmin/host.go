package nodeadmin

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ScorerHost interface {
	Pool() *pgxpool.Pool
	WithPostgresLow(ctx context.Context, fn func(context.Context) error) error
	ScorerConfig() ScorerConfig
	RegionCode() int16
	MultiRegionGlobal() bool
	UDPSyncInterval() time.Duration
	ScoringMetricsForRole(role string) []ScoringMetricDef
}
