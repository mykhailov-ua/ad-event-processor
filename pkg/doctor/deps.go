package doctor

import (
	"context"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/bidshard/ad-event-processor/internal/licensing"

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
}
