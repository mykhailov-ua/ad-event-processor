package controlplane

import (
	"context"
	"encoding/json"
	"fmt"

	"espx/internal/domain"
	db "espx/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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

type cohortSnapshotPayload struct {
	Version int64 `json:"version"`
}

func (s *Service) UpsertExperimentCohort(ctx context.Context, spec ExperimentCohortSpec) error {
	if s == nil || s.pool == nil {
		return fmt.Errorf("service unavailable")
	}
	if spec.ID == uuid.Nil || spec.Name == "" || spec.Salt == "" || len(spec.Variants) == 0 {
		return fmt.Errorf("invalid cohort spec")
	}
	variantsJSON, err := json.Marshal(spec.Variants)
	if err != nil {
		return fmt.Errorf("marshal cohort variants: %w", err)
	}

	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
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

		payloadBytes, err := json.Marshal(cohortSnapshotPayload{Version: 1})
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
		s.AuditLog(ctx, q, uuid.Nil, "UPDATE_COHORT_SNAPSHOT", "experiment", &spec.ID, map[string]any{
			"name":     spec.Name,
			"active":   spec.Active,
			"variants": len(spec.Variants),
		}, map[string]any{"outbox_event_id": ev.ID})
		return nil
	})
}

func (s *Service) publishRegistryFullSync(ctx context.Context) error {
	return s.publishCampaignUpdate(ctx, domain.RegistryFullSyncPayload)
}
