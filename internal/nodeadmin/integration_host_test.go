package nodeadmin

import (
	"context"
	"time"

	"ad-event-processor/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

type integrationScorerHost struct {
	pool *pgxpool.Pool
	cfg  *config.Config
}

func newIntegrationScorerHost(pool *pgxpool.Pool, cfg *config.Config) *integrationScorerHost {
	return &integrationScorerHost{pool: pool, cfg: cfg}
}

func (h *integrationScorerHost) Pool() *pgxpool.Pool { return h.pool }

func (h *integrationScorerHost) WithPostgresLow(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func (h *integrationScorerHost) ScorerConfig() ScorerConfig { return ScorerConfigFrom(h.cfg) }

func (h *integrationScorerHost) RegionCode() int16 {
	if h.cfg == nil {
		return 0
	}
	return int16(h.cfg.RegionCode)
}

func (h *integrationScorerHost) MultiRegionGlobal() bool {
	return h.cfg != nil && h.cfg.MultiRegionGlobal()
}

func (h *integrationScorerHost) UDPSyncInterval() time.Duration {
	if h.cfg == nil || h.cfg.UDPSyncIntervalMs <= 0 {
		return 10 * time.Second
	}
	return time.Duration(h.cfg.UDPSyncIntervalMs) * time.Millisecond
}

func (h *integrationScorerHost) ScoringMetricsForRole(role string) []ScoringMetricDef {
	return MetricsForRole(role)
}
