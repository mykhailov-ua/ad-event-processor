package doctor

import (
	"context"

	"espx/internal/config"
	"espx/internal/licensing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type ProbeDeps struct {
	Config             *config.Config
	Redis              func(context.Context) ([]redis.UniversalClient, error)
	CHPing             func(context.Context) error
	PGPool             func(context.Context) (*pgxpool.Pool, error)
	LicenseState       func() (licensing.LicenseState, bool)
	LicenseDiagnostics func() (licensing.LicenseDiagnostics, bool)
}
