package doctor

import (
	"context"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/edge"
	"ad-event-processor/internal/licensing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type ProbeDeps struct {
	Config             *config.Config
	Redis              func(context.Context) ([]redis.UniversalClient, error)
	CHPing             func(context.Context) error
	PGPool             func(context.Context) (*pgxpool.Pool, error)
	SlotMapFromPG      func(context.Context) (domain.OpsSlotMapResponse, error)
	SlotMapFromHTTP    func(context.Context, string) (domain.OpsSlotMapResponse, error)
	LicenseState       func() (licensing.LicenseState, bool)
	LicenseDiagnostics func() (licensing.LicenseDiagnostics, bool)
	XDPStatsReader     func(context.Context) (edge.Snapshot, error)
}
