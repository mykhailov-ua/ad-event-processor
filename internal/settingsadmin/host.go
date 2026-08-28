package settingsadmin

import (
	"context"

	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type Host interface {
	ErrValidation(msg string) error
	ActorUserID(ctx context.Context) uuid.UUID
	AuditLog(ctx context.Context, q db.Querier, adminID uuid.UUID, action, targetType string, targetID *uuid.UUID, changes, metadata any)
	AuditEmergencyBreaker(ctx context.Context, q db.Querier, adminID uuid.UUID, active bool, reason string)
	IsProtectedIP(ip string) bool
	BlacklistAutoTTLHours() int
	BlacklistFraudTTLHours() int
	RedisShards() []redis.UniversalClient
	SyncGlobalSetReplace(ctx context.Context, key string, members []any) error
	SyncGlobalConfig(ctx context.Context, settings map[string]string) error
	ReplicateConfigVersionFromPrimary(ctx context.Context) error
	NewBlockIPPreview(change BlockIPPreviewChange) (MutationPreview, error)
}
