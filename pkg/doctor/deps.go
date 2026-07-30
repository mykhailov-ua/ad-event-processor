package doctor

import (
	"context"

	"espx/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type ProbeDeps struct {
	Config *config.Config
	Redis  func(context.Context) ([]redis.UniversalClient, error)
	CHPing func(context.Context) error
	PGPool func(context.Context) (*pgxpool.Pool, error)
}
