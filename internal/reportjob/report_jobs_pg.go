package reportjob

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type claimedReportJob struct {
	id   uuid.UUID
	spec ReportJobSpec
}

func (r *ReportJobRunner) PgEnabled() bool {
	return r.pgEnabled()
}

func (r *ReportJobRunner) pgEnabled() bool {
	return r != nil && r.deps.Pool != nil
}

func (r *ReportJobRunner) createJobPG(ctx context.Context, spec ReportJobSpec, idempotencyKey string) (string, error) {
	if idempotencyKey != "" {
		existing, ok, err := r.lookupReportJobByIdempotency(ctx, idempotencyKey)
		if err != nil {
			return "", err
		}
		if ok {
			return existing, nil
		}
	}
	specJSON, err := json.Marshal(spec)
	if err != nil {
		return "", fmt.Errorf("marshal report job spec: %w", err)
	}
	var jobID uuid.UUID
	err = r.deps.Pool.QueryRow(ctx, `
INSERT INTO report_jobs (customer_id, report_key, spec, idempotency_key)
VALUES ($1, $2, $3, NULLIF($4, ''))
RETURNING id`,
		uuid.MustParse(spec.CustomerID),
		spec.ReportKey,
		specJSON,
		idempotencyKey,
	).Scan(&jobID)
	if err != nil {
		return "", fmt.Errorf("insert report job: %w", err)
	}
	return jobID.String(), nil
}

func (r *ReportJobRunner) lookupReportJobByIdempotency(ctx context.Context, idempotencyKey string) (string, bool, error) {
	var jobID uuid.UUID
	err := r.deps.Pool.QueryRow(ctx, `
SELECT id FROM report_jobs WHERE idempotency_key = $1`, idempotencyKey).Scan(&jobID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", false, nil
		}
		return "", false, err
	}
	return jobID.String(), true, nil
}

func (r *ReportJobRunner) getJobPG(ctx context.Context, jobID string) (ReportJobStatusDTO, bool, error) {
	parsed, err := uuid.Parse(jobID)
	if err != nil {
		return ReportJobStatusDTO{}, false, nil
	}
	var dto ReportJobStatusDTO
	var customerID uuid.UUID
	var specJSON []byte
	var errMsg *string
	var createdAt time.Time
	err = r.deps.Pool.QueryRow(ctx, `
SELECT id, customer_id, report_key, spec, status, COALESCE(bytes, 0), error_message, created_at
FROM report_jobs
WHERE id = $1`, parsed).Scan(
		&parsed, &customerID, &dto.ReportKey, &specJSON, &dto.Status, &dto.Bytes, &errMsg, &createdAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ReportJobStatusDTO{}, false, nil
		}
		return ReportJobStatusDTO{}, false, err
	}
	var spec ReportJobSpec
	if err := json.Unmarshal(specJSON, &spec); err != nil {
		return ReportJobStatusDTO{}, false, fmt.Errorf("decode report job spec: %w", err)
	}
	dto.ID = parsed.String()
	dto.JobID = dto.ID
	dto.CustomerID = customerID.String()
	dto.Format = spec.Format
	if errMsg != nil {
		dto.Error = *errMsg
	}
	dto.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	return dto, true, nil
}

func (r *ReportJobRunner) listJobsByCustomerPG(ctx context.Context, customerID string, limit int) ([]ReportJobStatusDTO, error) {
	cid, err := uuid.Parse(customerID)
	if err != nil {
		return nil, fmt.Errorf("invalid customer_id")
	}
	rows, err := r.deps.Pool.Query(ctx, `
SELECT id, customer_id, report_key, spec, status, COALESCE(bytes, 0), error_message, created_at
FROM report_jobs
WHERE customer_id = $1
ORDER BY created_at DESC
LIMIT $2`, cid, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]ReportJobStatusDTO, 0, limit)
	for rows.Next() {
		var jobID, rowCustomerID uuid.UUID
		var specJSON []byte
		var dto ReportJobStatusDTO
		var errMsg *string
		var createdAt time.Time
		if err := rows.Scan(&jobID, &rowCustomerID, &dto.ReportKey, &specJSON, &dto.Status, &dto.Bytes, &errMsg, &createdAt); err != nil {
			return nil, err
		}
		var spec ReportJobSpec
		if err := json.Unmarshal(specJSON, &spec); err != nil {
			return nil, err
		}
		dto.ID = jobID.String()
		dto.JobID = dto.ID
		dto.CustomerID = rowCustomerID.String()
		dto.Format = spec.Format
		if errMsg != nil {
			dto.Error = *errMsg
		}
		dto.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		out = append(out, dto)
	}
	return out, rows.Err()
}

func (r *ReportJobRunner) openDownloadPG(ctx context.Context, jobID string) (string, ReportJobStatusDTO, error) {
	dto, ok, err := r.getJobPG(ctx, jobID)
	if err != nil {
		return "", ReportJobStatusDTO{}, err
	}
	if !ok {
		return "", ReportJobStatusDTO{}, fmt.Errorf("job not found")
	}
	if dto.Status != JobStatusCompleted {
		return "", dto, fmt.Errorf("export not ready")
	}
	var filePath *string
	parsed, err := uuid.Parse(jobID)
	if err != nil {
		return "", dto, fmt.Errorf("invalid job id")
	}
	err = r.deps.Pool.QueryRow(ctx, `SELECT file_path FROM report_jobs WHERE id = $1`, parsed).Scan(&filePath)
	if err != nil {
		return "", dto, err
	}
	if filePath == nil || *filePath == "" {
		return "", dto, fmt.Errorf("export not ready")
	}
	return *filePath, dto, nil
}

func claimReportJobs(ctx context.Context, pool *pgxpool.Pool, limit int) ([]claimedReportJob, error) {
	var claimed []claimedReportJob
	err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
SELECT id, spec
FROM report_jobs
WHERE status = $1
ORDER BY created_at
LIMIT $2
FOR UPDATE SKIP LOCKED`, JobStatusPending, limit)
		if err != nil {
			return fmt.Errorf("claim report jobs: %w", err)
		}
		defer rows.Close()

		type pending struct {
			id       uuid.UUID
			specJSON []byte
		}
		var pendingJobs []pending
		for rows.Next() {
			var row pending
			if err := rows.Scan(&row.id, &row.specJSON); err != nil {
				return err
			}
			pendingJobs = append(pendingJobs, row)
		}
		if err := rows.Err(); err != nil {
			return err
		}
		for _, row := range pendingJobs {
			tag, err := tx.Exec(ctx, `
UPDATE report_jobs
SET status = $2, updated_at = NOW()
WHERE id = $1 AND status = $3`, row.id, JobStatusRunning, JobStatusPending)
			if err != nil {
				return err
			}
			if tag.RowsAffected() == 0 {
				continue
			}
			var spec ReportJobSpec
			if err := json.Unmarshal(row.specJSON, &spec); err != nil {
				return err
			}
			claimed = append(claimed, claimedReportJob{id: row.id, spec: spec})
		}
		return nil
	})
	return claimed, err
}

func completeReportJobPG(ctx context.Context, pool *pgxpool.Pool, jobID, filePath string, bytes int64) error {
	parsed, err := uuid.Parse(jobID)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `
UPDATE report_jobs
SET status = $2, file_path = $3, bytes = $4, error_message = NULL, updated_at = NOW()
WHERE id = $1`, parsed, JobStatusCompleted, filePath, bytes)
	return err
}

func failReportJobPG(ctx context.Context, pool *pgxpool.Pool, jobID string, errMsg string) error {
	parsed, err := uuid.Parse(jobID)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `
UPDATE report_jobs
SET status = $2, error_message = $3, updated_at = NOW()
WHERE id = $1`, parsed, JobStatusFailed, errMsg)
	return err
}

func (r *ReportJobRunner) cancelJobPG(ctx context.Context, jobID string) (ReportJobStatusDTO, bool, error) {
	if r == nil || r.deps.Pool == nil {
		return ReportJobStatusDTO{}, false, fmt.Errorf("report job store unavailable")
	}
	parsed, err := uuid.Parse(jobID)
	if err != nil {
		return ReportJobStatusDTO{}, false, err
	}
	tag, err := r.deps.Pool.Exec(ctx, `
UPDATE report_jobs
SET status = $2, updated_at = NOW()
WHERE id = $1 AND status = $3`, parsed, JobStatusCancelled, JobStatusPending)
	if err != nil {
		return ReportJobStatusDTO{}, false, err
	}
	if tag.RowsAffected() == 0 {
		dto, ok, gerr := r.getJobPG(ctx, jobID)
		if gerr != nil {
			return ReportJobStatusDTO{}, false, gerr
		}
		if !ok {
			return ReportJobStatusDTO{}, false, nil
		}
		return dto, false, fmt.Errorf("job cannot be cancelled")
	}
	dto, ok, err := r.getJobPG(ctx, jobID)
	return dto, ok, err
}
