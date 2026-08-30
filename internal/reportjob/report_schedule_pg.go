package reportjob

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ReportScheduleDTO struct {
	ID         string          `json:"id"`
	CustomerID string          `json:"customer_id"`
	ReportKey  string          `json:"report_key"`
	Format     string          `json:"format"`
	CronExpr   string          `json:"cron_expr"`
	Spec       json.RawMessage `json:"spec"`
	Enabled    bool            `json:"enabled"`
	NextRunAt  string          `json:"next_run_at"`
	LastRunAt  string          `json:"last_run_at,omitempty"`
	LastJobID  string          `json:"last_job_id,omitempty"`
	CreatedAt  string          `json:"created_at"`
	UpdatedAt  string          `json:"updated_at"`
}

type CreateReportScheduleRequest struct {
	CustomerID string          `json:"customer_id"`
	ReportKey  string          `json:"report_key"`
	Format     string          `json:"format"`
	CronExpr   string          `json:"cron_expr"`
	Spec       json.RawMessage `json:"spec"`
	Enabled    *bool           `json:"enabled"`
}

type UpdateReportScheduleRequest struct {
	ReportKey string          `json:"report_key"`
	Format    string          `json:"format"`
	CronExpr  string          `json:"cron_expr"`
	Spec      json.RawMessage `json:"spec"`
	Enabled   *bool           `json:"enabled"`
}

type reportScheduleRow struct {
	id         uuid.UUID
	customerID uuid.UUID
	reportKey  string
	format     string
	cronExpr   string
	specJSON   []byte
	enabled    bool
	nextRunAt  time.Time
}

type reportScheduleRangeSpec struct {
	From           string `json:"from"`
	To             string `json:"to"`
	FromOffsetDays int    `json:"from_offset_days"`
	ToOffsetDays   int    `json:"to_offset_days"`
}

func insertReportSchedule(ctx context.Context, pool *pgxpool.Pool, req CreateReportScheduleRequest) (ReportScheduleDTO, error) {
	if err := validateReportCronExpr(req.CronExpr); err != nil {
		return ReportScheduleDTO{}, fmt.Errorf("invalid cron_expr")
	}
	nextRun, err := nextReportCronRun(req.CronExpr, time.Now().UTC())
	if err != nil {
		return ReportScheduleDTO{}, err
	}
	format := req.Format
	if format == "" {
		format = "csv"
	}
	spec := req.Spec
	if len(spec) == 0 {
		spec = json.RawMessage(`{}`)
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	var id uuid.UUID
	var createdAt, updatedAt time.Time
	err = pool.QueryRow(ctx, `
INSERT INTO report_schedules (customer_id, report_key, format, cron_expr, spec, enabled, next_run_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, created_at, updated_at`,
		uuid.MustParse(req.CustomerID),
		req.ReportKey,
		format,
		req.CronExpr,
		spec,
		enabled,
		nextRun,
	).Scan(&id, &createdAt, &updatedAt)
	if err != nil {
		return ReportScheduleDTO{}, fmt.Errorf("insert report schedule: %w", err)
	}
	return ReportScheduleDTO{
		ID:         id.String(),
		CustomerID: req.CustomerID,
		ReportKey:  req.ReportKey,
		Format:     format,
		CronExpr:   req.CronExpr,
		Spec:       spec,
		Enabled:    enabled,
		NextRunAt:  nextRun.UTC().Format(time.RFC3339),
		CreatedAt:  createdAt.UTC().Format(time.RFC3339),
		UpdatedAt:  updatedAt.UTC().Format(time.RFC3339),
	}, nil
}

func getReportSchedule(ctx context.Context, pool *pgxpool.Pool, id string) (ReportScheduleDTO, error) {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return ReportScheduleDTO{}, pgx.ErrNoRows
	}
	return scanReportSchedule(ctx, pool, parsed)
}

func scanReportSchedule(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (ReportScheduleDTO, error) {
	var dto ReportScheduleDTO
	var customerID uuid.UUID
	var specJSON []byte
	var nextRunAt time.Time
	var lastRunAt *time.Time
	var lastJobID *uuid.UUID
	var createdAt, updatedAt time.Time
	err := pool.QueryRow(ctx, `
SELECT id, customer_id, report_key, format, cron_expr, spec, enabled, next_run_at, last_run_at, last_job_id, created_at, updated_at
FROM report_schedules
WHERE id = $1`, id).Scan(
		&id, &customerID, &dto.ReportKey, &dto.Format, &dto.CronExpr, &specJSON, &dto.Enabled,
		&nextRunAt, &lastRunAt, &lastJobID, &createdAt, &updatedAt,
	)
	if err != nil {
		return ReportScheduleDTO{}, err
	}
	dto.ID = id.String()
	dto.CustomerID = customerID.String()
	dto.Spec = specJSON
	dto.NextRunAt = nextRunAt.UTC().Format(time.RFC3339)
	if lastRunAt != nil {
		dto.LastRunAt = lastRunAt.UTC().Format(time.RFC3339)
	}
	if lastJobID != nil {
		dto.LastJobID = lastJobID.String()
	}
	dto.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	dto.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)
	return dto, nil
}

func listReportSchedules(ctx context.Context, pool *pgxpool.Pool, customerID string) ([]ReportScheduleDTO, error) {
	cid, err := uuid.Parse(customerID)
	if err != nil {
		return nil, fmt.Errorf("invalid customer_id")
	}
	rows, err := pool.Query(ctx, `
SELECT id, customer_id, report_key, format, cron_expr, spec, enabled, next_run_at, last_run_at, last_job_id, created_at, updated_at
FROM report_schedules
WHERE customer_id = $1
ORDER BY created_at DESC`, cid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]ReportScheduleDTO, 0, 8)
	for rows.Next() {
		var dto ReportScheduleDTO
		var id, rowCustomerID uuid.UUID
		var specJSON []byte
		var nextRunAt time.Time
		var lastRunAt *time.Time
		var lastJobID *uuid.UUID
		var createdAt, updatedAt time.Time
		if err := rows.Scan(
			&id, &rowCustomerID, &dto.ReportKey, &dto.Format, &dto.CronExpr, &specJSON, &dto.Enabled,
			&nextRunAt, &lastRunAt, &lastJobID, &createdAt, &updatedAt,
		); err != nil {
			return nil, err
		}
		dto.ID = id.String()
		dto.CustomerID = rowCustomerID.String()
		dto.Spec = specJSON
		dto.NextRunAt = nextRunAt.UTC().Format(time.RFC3339)
		if lastRunAt != nil {
			dto.LastRunAt = lastRunAt.UTC().Format(time.RFC3339)
		}
		if lastJobID != nil {
			dto.LastJobID = lastJobID.String()
		}
		dto.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		dto.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)
		out = append(out, dto)
	}
	return out, rows.Err()
}

func updateReportSchedule(ctx context.Context, pool *pgxpool.Pool, id string, req UpdateReportScheduleRequest) (ReportScheduleDTO, error) {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return ReportScheduleDTO{}, pgx.ErrNoRows
	}
	existing, err := scanReportSchedule(ctx, pool, parsed)
	if err != nil {
		return ReportScheduleDTO{}, err
	}
	if err := validateReportCronExpr(req.CronExpr); err != nil {
		return ReportScheduleDTO{}, fmt.Errorf("invalid cron_expr")
	}
	nextRun, err := nextReportCronRun(req.CronExpr, time.Now().UTC())
	if err != nil {
		return ReportScheduleDTO{}, err
	}
	format := req.Format
	if format == "" {
		format = "csv"
	}
	spec := req.Spec
	if len(spec) == 0 {
		spec = json.RawMessage(`{}`)
	}
	enabled := existing.Enabled
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	tag, err := pool.Exec(ctx, `
UPDATE report_schedules
SET report_key = $2, format = $3, cron_expr = $4, spec = $5, enabled = $6, next_run_at = $7, updated_at = NOW()
WHERE id = $1`,
		parsed, req.ReportKey, format, req.CronExpr, spec, enabled, nextRun,
	)
	if err != nil {
		return ReportScheduleDTO{}, err
	}
	if tag.RowsAffected() == 0 {
		return ReportScheduleDTO{}, pgx.ErrNoRows
	}
	return scanReportSchedule(ctx, pool, parsed)
}

func deleteReportSchedule(ctx context.Context, pool *pgxpool.Pool, id string) error {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return pgx.ErrNoRows
	}
	tag, err := pool.Exec(ctx, `DELETE FROM report_schedules WHERE id = $1`, parsed)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func claimDueReportSchedules(ctx context.Context, pool *pgxpool.Pool, limit int) ([]reportScheduleRow, error) {
	// Txn claims due rows (SKIP LOCKED) and advances next_run_at before enqueue to prevent double-fire.
	var claimed []reportScheduleRow
	err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
SELECT id, customer_id, report_key, format, cron_expr, spec, enabled, next_run_at
FROM report_schedules
WHERE enabled = TRUE AND next_run_at <= NOW()
ORDER BY next_run_at
LIMIT $1
FOR UPDATE SKIP LOCKED`, limit)
		if err != nil {
			return err
		}
		defer rows.Close()

		type pending struct {
			row      reportScheduleRow
			nextRun  time.Time
			nextNext time.Time
		}
		var pendingRows []pending
		for rows.Next() {
			var row reportScheduleRow
			if err := rows.Scan(&row.id, &row.customerID, &row.reportKey, &row.format, &row.cronExpr, &row.specJSON, &row.enabled, &row.nextRunAt); err != nil {
				return err
			}
			nextNext, err := nextReportCronRun(row.cronExpr, row.nextRunAt)
			if err != nil {
				return err
			}
			pendingRows = append(pendingRows, pending{row: row, nextRun: row.nextRunAt, nextNext: nextNext})
		}
		if err := rows.Err(); err != nil {
			return err
		}
		for _, item := range pendingRows {
			tag, err := tx.Exec(ctx, `
UPDATE report_schedules
SET next_run_at = $2, last_run_at = NOW(), updated_at = NOW()
WHERE id = $1 AND next_run_at = $3`,
				item.row.id, item.nextNext, item.nextRun,
			)
			if err != nil {
				return err
			}
			if tag.RowsAffected() == 0 {
				continue
			}
			claimed = append(claimed, item.row)
		}
		return nil
	})
	return claimed, err
}

func markReportScheduleJob(ctx context.Context, pool *pgxpool.Pool, scheduleID, jobID string) error {
	sid, err := uuid.Parse(scheduleID)
	if err != nil {
		return err
	}
	jid, err := uuid.Parse(jobID)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `
UPDATE report_schedules SET last_job_id = $2, updated_at = NOW() WHERE id = $1`, sid, jid)
	return err
}

func buildReportJobSpecFromSchedule(row reportScheduleRow) (ReportJobSpec, string, error) {
	var rangeSpec reportScheduleRangeSpec
	if len(row.specJSON) > 0 {
		_ = json.Unmarshal(row.specJSON, &rangeSpec)
	}
	now := time.Now().UTC()
	to := now.Add(time.Duration(rangeSpec.ToOffsetDays) * 24 * time.Hour)
	fromDays := rangeSpec.FromOffsetDays
	if fromDays <= 0 {
		fromDays = 7 // default lookback when spec omits from_offset_days
	}
	from := now.Add(-time.Duration(fromDays) * 24 * time.Hour)
	if rangeSpec.From != "" {
		parsed, err := time.Parse(time.RFC3339, rangeSpec.From)
		if err != nil {
			return ReportJobSpec{}, "", fmt.Errorf("invalid schedule spec.from")
		}
		from = parsed.UTC()
	}
	if rangeSpec.To != "" {
		parsed, err := time.Parse(time.RFC3339, rangeSpec.To)
		if err != nil {
			return ReportJobSpec{}, "", fmt.Errorf("invalid schedule spec.to")
		}
		to = parsed.UTC()
	}
	format := row.format
	if format == "" {
		format = "csv"
	}
	// PG report_jobs.idempotency_key: one job per schedule fired slot; replays return same job id.
	idem := fmt.Sprintf("schedule:%s:%s", row.id.String(), row.nextRunAt.UTC().Format("2006-01-02T15:04"))
	return ReportJobSpec{
		CustomerID: row.customerID.String(),
		ReportKey:  row.reportKey,
		From:       from.Format(time.RFC3339),
		To:         to.Format(time.RFC3339),
		Format:     format,
	}, idem, nil
}
