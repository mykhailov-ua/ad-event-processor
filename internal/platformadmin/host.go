package platformadmin

import (
	"context"

	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/pkg/platformconfig"

	"github.com/google/uuid"
)

type Host interface {
	VerifyInstallToken(token string) error
	ErrValidation(msg string) error
	ActorUserID(ctx context.Context) uuid.UUID
	SaveBootstrapEula(ctx context.Context, q db.Querier, version, acceptedBy string) error
	AuditBootstrap(ctx context.Context, q db.Querier, adminID uuid.UUID, cfg platformconfig.Config)
	AuditUpdate(ctx context.Context, q db.Querier, adminID uuid.UUID, before, after platformconfig.Config, restartRequired []string)
	AuditApply(ctx context.Context, q db.Querier, adminID uuid.UUID, writtenPath string)
	SyncEdgeExpose(ctx context.Context, cfg platformconfig.Config) error
}
