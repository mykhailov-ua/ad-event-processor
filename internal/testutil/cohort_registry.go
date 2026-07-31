package testutil

import (
	"context"

	"espx/internal/ingestion"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	db "espx/internal/domain/db"
)

type CohortRegistry interface {
	SyncCohorts(ctx context.Context) error
	AssignExperiment(experimentID uuid.UUID, subjectID string) (variantID string, flags map[string]string, ok bool)
	ExperimentCount() int
}

func NewCohortRegistry(pool *pgxpool.Pool) CohortRegistry {
	r := ingestion.NewRegistry(db.New(pool))
	r.SetPool(pool)
	return r
}
