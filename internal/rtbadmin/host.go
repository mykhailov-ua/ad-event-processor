package rtbadmin

import (
	"context"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Host interface {
	Pool() *pgxpool.Pool
	Config() *config.Config
	ErrValidation(msg string) error
	ActorUserID(ctx context.Context) uuid.UUID
	AuditCreateRtbDeal(ctx context.Context, q db.Querier, adminID uuid.UUID, dealID string)
	AuditUpdateRtbDeal(ctx context.Context, q db.Querier, adminID uuid.UUID, id int64, dealID string)
	AuditDeleteRtbDeal(ctx context.Context, q db.Querier, adminID uuid.UUID, id int64, dealID string)
	EnqueueRtbCatalogReload(ctx context.Context, q db.Querier, trigger string) error
	UpdateSettings(ctx context.Context, patch map[string]string) error
	GetSettings(ctx context.Context) (map[string]string, error)
	SimulateBidShade(ctx context.Context, in domain.RtbBidShadeInput) (domain.RtbBidShadeOutput, error)
}
