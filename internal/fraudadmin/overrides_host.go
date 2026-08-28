package fraudadmin

import (
	"context"

	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OverridesHost interface {
	OverridesPool() *pgxpool.Pool
	OverrideActorID(ctx context.Context) uuid.UUID
	OverrideAuditLog(ctx context.Context, q db.Querier, adminID uuid.UUID, action, targetType string, targetID *uuid.UUID, changes, metadata any)
	OverrideHashIP(ip string) ([16]byte, error)
	OverrideEnqueueClearBoost(ctx context.Context, q db.Querier, campaignID string) error
	OverrideEnqueueBlacklistRemove(ctx context.Context, q db.Querier, ip string) error
}
