package campaign

import (
	"context"
	"encoding/json"
	"fmt"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CohortVariantSpec struct {
	ID     string            `json:"id"`
	Weight uint32            `json:"weight"`
	Flags  map[string]string `json:"flags,omitempty"`
}

type ExperimentCohortSpec struct {
	ID       uuid.UUID           `json:"id"`
	Name     string              `json:"name"`
	Active   bool                `json:"active"`
	Salt     string              `json:"salt"`
	Variants []CohortVariantSpec `json:"variants"`
}

func UpsertExperimentCohort(ctx context.Context, host ExperimentsHost, spec ExperimentCohortSpec) error {
	if host == nil || host.Pool() == nil {
		return fmt.Errorf("service unavailable")
	}
	if spec.ID == uuid.Nil || spec.Name == "" || spec.Salt == "" || len(spec.Variants) == 0 {
		return fmt.Errorf("invalid cohort spec")
	}
	variantsJSON, err := json.Marshal(spec.Variants)
	if err != nil {
		return fmt.Errorf("marshal cohort variants: %w", err)
	}

	return pgx.BeginFunc(ctx, host.Pool(), func(tx pgx.Tx) error {
		q := db.New(tx)
		_, err := q.UpsertExperimentCohort(ctx, db.UpsertExperimentCohortParams{
			ID:       domain.ToUUID(spec.ID),
			Name:     spec.Name,
			Active:   spec.Active,
			Salt:     spec.Salt,
			Variants: variantsJSON,
		})
		if err != nil {
			return err
		}

		payloadBytes, err := host.CohortSnapshotOutboxPayload()
		if err != nil {
			return err
		}
		ev, err := q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
			EventType: "UPDATE_COHORT_SNAPSHOT",
			Payload:   payloadBytes,
		})
		if err != nil {
			return err
		}
		host.AuditCohortSnapshotChange(ctx, q, spec.ID, ExperimentCohortAuditChange{
			Name:     spec.Name,
			Active:   spec.Active,
			Variants: len(spec.Variants),
		}, ev.ID)
		return nil
	})
}

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
