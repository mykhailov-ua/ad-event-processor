package governance

import (
	"context"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Host interface {
	Pool() *pgxpool.Pool
	RedisShards() []redis.UniversalClient
	Sharder() domain.Sharder
	Config() *config.Config
	AuditLog(ctx context.Context, q db.Querier, adminID uuid.UUID, action, targetType string, targetID *uuid.UUID, changes, metadata any)
}
