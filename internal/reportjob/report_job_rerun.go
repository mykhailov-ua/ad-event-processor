package reportjob

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func (r *ReportJobRunner) LoadJobSpec(ctx context.Context, jobID string) (ReportJobSpec, bool, error) {
	if r == nil {
		return ReportJobSpec{}, false, fmt.Errorf("report job runner unavailable")
	}
	if r.pgEnabled() {
		return loadJobSpecPG(ctx, r.deps.Pool, jobID)
	}
	r.mu.RLock()
	rec, ok := r.jobs[jobID]
	r.mu.RUnlock()
	if !ok {
		return ReportJobSpec{}, false, nil
	}
	return rec.spec, true, nil
}

func loadJobSpecPG(ctx context.Context, pool *pgxpool.Pool, jobID string) (ReportJobSpec, bool, error) {
	if pool == nil {
		return ReportJobSpec{}, false, fmt.Errorf("report job store unavailable")
	}
	parsed, err := uuid.Parse(jobID)
	if err != nil {
		return ReportJobSpec{}, false, nil
	}
	var spec ReportJobSpec
	var specJSON []byte
	err = pool.QueryRow(ctx, `SELECT spec FROM report_jobs WHERE id = $1`, parsed).Scan(&specJSON)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ReportJobSpec{}, false, nil
		}
		return ReportJobSpec{}, false, err
	}
	if err := json.Unmarshal(specJSON, &spec); err != nil {
		return ReportJobSpec{}, false, fmt.Errorf("decode report job spec: %w", err)
	}
	return spec, true, nil
}

func (r *ReportJobRunner) RerunJob(ctx context.Context, jobID string, idempotencyKey string) (string, error) {
	spec, ok, err := r.LoadJobSpec(ctx, jobID)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("job not found")
	}
	return r.CreateJob(ctx, spec, idempotencyKey)
}
