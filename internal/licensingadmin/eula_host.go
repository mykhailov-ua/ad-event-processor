package licensingadmin

import (
	"context"

	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EulaHost interface {
	EulaPool() *pgxpool.Pool
	EulaActorID(ctx context.Context) uuid.UUID
	EulaAuditAccept(ctx context.Context, q db.Querier, adminID uuid.UUID, version, acceptedBy string)
}
