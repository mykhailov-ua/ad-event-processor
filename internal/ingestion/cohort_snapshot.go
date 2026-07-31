package ingestion

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"espx/internal/domain"
	db "espx/internal/domain/db"

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
		cohort, convErr := experimentCohortFromRow(row)
		if convErr != nil {
			slog.Warn("skip invalid experiment cohort row", "id", uuid.UUID(row.ID.Bytes), "error", convErr)
			continue
		}
		byID[cohort.ID] = cohort
	}
	r.cohorts.Store(&cohortRegistrySnapshot{byID: byID})
	return nil
}

func experimentCohortFromRow(row db.ExperimentCohort) (domain.ExperimentCohort, error) {
	id := uuid.UUID(row.ID.Bytes)
	var variants []cohortVariantDTO
	if err := json.Unmarshal(row.Variants, &variants); err != nil {
		return domain.ExperimentCohort{}, fmt.Errorf("decode variants: %w", err)
	}
	out := domain.ExperimentCohort{
		ID:       id,
		Name:     row.Name,
		Salt:     row.Salt,
		Variants: make([]domain.CohortVariant, 0, len(variants)),
	}
	for _, v := range variants {
		if v.ID == "" || v.Weight == 0 {
			continue
		}
		out.Variants = append(out.Variants, domain.CohortVariant{
			ID:     v.ID,
			Weight: v.Weight,
			Flags:  v.Flags,
		})
	}
	if len(out.Variants) == 0 {
		return domain.ExperimentCohort{}, fmt.Errorf("no valid variants")
	}
	return out, nil
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
