package ingestion

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/bidshard/ad-event-processor/internal/domain"
	db "github.com/bidshard/ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
)

type cohortVariantDTO struct {
	ID     string            `json:"id"`
	Weight uint32            `json:"weight"`
	Flags  map[string]string `json:"flags,omitempty"`
}

type cohortRegistrySnapshot struct {
	byID map[uuid.UUID]domain.ExperimentCohort
}

func (r *Registry) SyncCohorts(ctx context.Context) error {
	if r == nil || r.repo == nil {
		return nil
	}
	listFn, ok := r.repo.(interface {
		ListActiveExperimentCohorts(context.Context) ([]db.ExperimentCohort, error)
	})
	if !ok {
		return nil
	}
	rows, err := listFn.ListActiveExperimentCohorts(ctx)
	if err != nil {
		return err
	}
	byID := make(map[uuid.UUID]domain.ExperimentCohort, len(rows))
	for _, row := range rows {
		id := uuid.UUID(row.ID.Bytes)
		var variants []cohortVariantDTO
		if err := json.Unmarshal(row.Variants, &variants); err != nil {
			slog.Warn("skip invalid experiment cohort row", "id", id, "error", err)
			continue
		}
		cohort := domain.ExperimentCohort{
			ID:       id,
			Name:     row.Name,
			Salt:     row.Salt,
			Variants: make([]domain.CohortVariant, 0, len(variants)),
		}
		for _, v := range variants {
			if v.ID == "" || v.Weight == 0 {
				continue
			}
			cohort.Variants = append(cohort.Variants, domain.CohortVariant{
				ID:     v.ID,
				Weight: v.Weight,
				Flags:  v.Flags,
			})
		}
		if len(cohort.Variants) == 0 {
			slog.Warn("skip invalid experiment cohort row", "id", id, "error", "no valid variants")
			continue
		}
		byID[cohort.ID] = cohort
	}
	r.cohorts.Store(&cohortRegistrySnapshot{byID: byID})
	return nil
}

func (r *Registry) cohortSnapshot() *cohortRegistrySnapshot {
	if r == nil {
		return &cohortRegistrySnapshot{}
	}
	v, ok := r.cohorts.Load().(*cohortRegistrySnapshot)
	if !ok || v == nil {
		return &cohortRegistrySnapshot{}
	}
	return v
}

func (r *Registry) AssignExperiment(experimentID uuid.UUID, subjectID string) (variantID string, flags map[string]string, ok bool) {
	if r == nil || subjectID == "" {
		return "", nil, false
	}
	cohort, found := r.cohortSnapshot().byID[experimentID]
	if !found {
		return "", nil, false
	}
	variantID, flags = AssignCohortVariant(cohort.Salt, subjectID, cohort.Variants)
	return variantID, flags, variantID != ""
}

func (r *Registry) ExperimentCount() int {
	return len(r.cohortSnapshot().byID)
}
