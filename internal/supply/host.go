package supply

import (
	"context"

	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
)

type Host interface {
	ActorUserID(ctx context.Context) uuid.UUID
	AuditLog(ctx context.Context, q db.Querier, adminID uuid.UUID, action, targetType string, targetID *uuid.UUID, changes, metadata any)
	EnqueueSupplyFilesUpdate(ctx context.Context, q db.Querier, trigger string) error
	SupplyExportPath() string
	ErrValidation(msg string) error
}
