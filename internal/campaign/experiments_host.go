package campaign

import (
	"context"

	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ExperimentsHost interface {
	Pool() *pgxpool.Pool
	CohortSnapshotOutboxPayload() ([]byte, error)
	AuditCohortSnapshotChange(ctx context.Context, q db.Querier, experimentID uuid.UUID, change ExperimentCohortAuditChange, outboxEventID int64)
}

type ExperimentCohortAuditChange struct {
	Name     string `json:"name"`
	Active   bool   `json:"active"`
	Variants int    `json:"variants"`
}
