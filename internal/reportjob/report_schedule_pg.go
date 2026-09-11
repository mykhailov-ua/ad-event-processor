package reportjob

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ReportScheduleDTO struct {
	ID                 string                   `json:"id"`
	CustomerID         string                   `json:"customer_id"`
	ReportKey          string                   `json:"report_key"`
	Format             string                   `json:"format"`
	Destination        string                   `json:"destination,omitempty"`
	OwnerUserID        string                   `json:"owner_user_id,omitempty"`
	GoogleSheet        ReportJobGoogleSheetSpec `json:"google_sheet,omitempty"`
	Notify             ReportJobNotifySpec      `json:"notify,omitempty"`
	CronExpr           string                   `json:"cron_expr"`
	Spec               json.RawMessage          `json:"spec"`
	Enabled            bool                     `json:"enabled"`
	NextRunAt          string                   `json:"next_run_at"`
	LastRunAt          string                   `json:"last_run_at,omitempty"`
	LastJobID          string                   `json:"last_job_id,omitempty"`
	LastRunStatus      string                   `json:"last_run_status,omitempty"`
	LastRunErrorPublic string                   `json:"last_run_error_public,omitempty"`
	CreatedAt          string                   `json:"created_at"`
	UpdatedAt          string                   `json:"updated_at"`
}

type CreateReportScheduleRequest struct {
	CustomerID  string                   `json:"customer_id"`
	ReportKey   string                   `json:"report_key"`
	Format      string                   `json:"format"`
	Destination string                   `json:"destination,omitempty"`
	OwnerUserID string                   `json:"owner_user_id,omitempty"`
	GoogleSheet ReportJobGoogleSheetSpec `json:"google_sheet,omitempty"`
	Notify      ReportJobNotifySpec      `json:"notify,omitempty"`
	CronExpr    string                   `json:"cron_expr"`
	Spec        json.RawMessage          `json:"spec"`
	Enabled     *bool                    `json:"enabled"`
}

type UpdateReportScheduleRequest struct {
	ReportKey   string                   `json:"report_key"`
	Format      string                   `json:"format"`
	Destination string                   `json:"destination,omitempty"`
	OwnerUserID string                   `json:"owner_user_id,omitempty"`
	GoogleSheet ReportJobGoogleSheetSpec `json:"google_sheet,omitempty"`
	Notify      ReportJobNotifySpec      `json:"notify,omitempty"`
	CronExpr    string                   `json:"cron_expr"`
	Spec        json.RawMessage          `json:"spec"`
	Enabled     *bool                    `json:"enabled"`
}

type reportScheduleRow struct {
	id          uuid.UUID
	customerID  uuid.UUID
	reportKey   string
	format      string
	destination string
	ownerUserID string
	cronExpr    string
	specJSON    []byte
	enabled     bool
	nextRunAt   time.Time
}

func normalizeReportScheduleDestination(destination string) string {
	destination = strings.TrimSpace(destination)
	if destination == "" {
		return "download"
	}
	return destination
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
	destination := normalizeReportScheduleDestination(req.Destination)
	spec, err := mergeReportScheduleSpec(req.Spec, req.GoogleSheet, req.Notify)
	if err != nil {
		return ReportScheduleDTO{}, fmt.Errorf("merge schedule spec: %w", err)
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	var ownerUserID *uuid.UUID
	if strings.TrimSpace(req.OwnerUserID) != "" {
		parsedOwner, parseErr := uuid.Parse(strings.TrimSpace(req.OwnerUserID))
		if parseErr != nil {
			return ReportScheduleDTO{}, fmt.Errorf("invalid owner_user_id")
		}
		ownerUserID = &parsedOwner
	}
	var id uuid.UUID
	var createdAt, updatedAt time.Time
	err = pool.QueryRow(ctx, `
INSERT INTO report_schedules (customer_id, report_key, format, destination, owner_user_id, cron_expr, spec, enabled, next_run_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id, created_at, updated_at`,
		uuid.MustParse(req.CustomerID),
		req.ReportKey,
		format,
		destination,
		ownerUserID,
		req.CronExpr,
		spec,
		enabled,
		nextRun,
	).Scan(&id, &createdAt, &updatedAt)
	if err != nil {
		return ReportScheduleDTO{}, fmt.Errorf("insert report schedule: %w", err)
	}
	dto := ReportScheduleDTO{
		ID:          id.String(),
		CustomerID:  req.CustomerID,
		ReportKey:   req.ReportKey,
		Format:      format,
		Destination: destination,
		OwnerUserID: strings.TrimSpace(req.OwnerUserID),
		GoogleSheet: req.GoogleSheet,
		Notify:      req.Notify,
		CronExpr:    req.CronExpr,
		Spec:        spec,
		Enabled:     enabled,
		NextRunAt:   nextRun.UTC().Format(time.RFC3339),
		CreatedAt:   createdAt.UTC().Format(time.RFC3339),
		UpdatedAt:   updatedAt.UTC().Format(time.RFC3339),
	}
	populateReportScheduleDTOFromSpec(&dto, spec)
	return dto, nil
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
	var ownerUserID *uuid.UUID
	var lastRunStatus, lastRunErrorPublic *string
	var createdAt, updatedAt time.Time
	err := pool.QueryRow(ctx, `
SELECT id, customer_id, report_key, format, destination, owner_user_id, cron_expr, spec, enabled, next_run_at,
       last_run_at, last_job_id, last_run_status, last_run_error_public, created_at, updated_at
FROM report_schedules
WHERE id = $1`, id).Scan(
		&id, &customerID, &dto.ReportKey, &dto.Format, &dto.Destination, &ownerUserID, &dto.CronExpr, &specJSON, &dto.Enabled,
		&nextRunAt, &lastRunAt, &lastJobID, &lastRunStatus, &lastRunErrorPublic, &createdAt, &updatedAt,
	)
	if err != nil {
		return ReportScheduleDTO{}, err
	}
	dto.ID = id.String()
	dto.CustomerID = customerID.String()
	dto.Spec = specJSON
	if dto.Destination == "" {
		dto.Destination = "download"
	}
	if ownerUserID != nil {
		dto.OwnerUserID = ownerUserID.String()
	}
	populateReportScheduleDTOFromSpec(&dto, specJSON)
	dto.NextRunAt = nextRunAt.UTC().Format(time.RFC3339)
	if lastRunAt != nil {
		dto.LastRunAt = lastRunAt.UTC().Format(time.RFC3339)
	}
	if lastJobID != nil {
		dto.LastJobID = lastJobID.String()
	}
	if lastRunStatus != nil {
		dto.LastRunStatus = *lastRunStatus
	}
	if lastRunErrorPublic != nil {
		dto.LastRunErrorPublic = *lastRunErrorPublic
	}
	dto.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	dto.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)
	enrichReportScheduleFromLastJob(ctx, pool, &dto)
	return dto, nil
}

func listReportSchedules(ctx context.Context, pool *pgxpool.Pool, customerID string) ([]ReportScheduleDTO, error) {
	cid, err := uuid.Parse(customerID)
	if err != nil {
		return nil, fmt.Errorf("invalid customer_id")
	}
	rows, err := pool.Query(ctx, `
SELECT id, customer_id, report_key, format, destination, owner_user_id, cron_expr, spec, enabled, next_run_at,
       last_run_at, last_job_id, last_run_status, last_run_error_public, created_at, updated_at
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
		var lastRunStatus, lastRunErrorPublic *string
		var ownerUserID *uuid.UUID
		if err := rows.Scan(
			&id, &rowCustomerID, &dto.ReportKey, &dto.Format, &dto.Destination, &ownerUserID, &dto.CronExpr, &specJSON, &dto.Enabled,
			&nextRunAt, &lastRunAt, &lastJobID, &lastRunStatus, &lastRunErrorPublic, &createdAt, &updatedAt,
		); err != nil {
			return nil, err
		}
		dto.ID = id.String()
		dto.CustomerID = rowCustomerID.String()
		dto.Spec = specJSON
		if dto.Destination == "" {
			dto.Destination = "download"
		}
		if ownerUserID != nil {
			dto.OwnerUserID = ownerUserID.String()
		}
		populateReportScheduleDTOFromSpec(&dto, specJSON)
		dto.NextRunAt = nextRunAt.UTC().Format(time.RFC3339)
		if lastRunAt != nil {
			dto.LastRunAt = lastRunAt.UTC().Format(time.RFC3339)
		}
		if lastJobID != nil {
			dto.LastJobID = lastJobID.String()
		}
		if lastRunStatus != nil {
			dto.LastRunStatus = *lastRunStatus
		}
		if lastRunErrorPublic != nil {
			dto.LastRunErrorPublic = *lastRunErrorPublic
		}
		dto.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		dto.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)
		enrichReportScheduleFromLastJob(ctx, pool, &dto)
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
	destination := existing.Destination
	if strings.TrimSpace(req.Destination) != "" {
		destination = normalizeReportScheduleDestination(req.Destination)
	}
	spec, err := mergeReportScheduleSpec(req.Spec, req.GoogleSheet, req.Notify)
	if err != nil {
		return ReportScheduleDTO{}, fmt.Errorf("merge schedule spec: %w", err)
	}
	enabled := existing.Enabled
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	ownerUserID := strings.TrimSpace(req.OwnerUserID)
	if ownerUserID == "" {
		ownerUserID = existing.OwnerUserID
	}
	var ownerUUID *uuid.UUID
	if ownerUserID != "" {
		parsedOwner, parseErr := uuid.Parse(ownerUserID)
		if parseErr != nil {
			return ReportScheduleDTO{}, fmt.Errorf("invalid owner_user_id")
		}
		ownerUUID = &parsedOwner
	}
	tag, err := pool.Exec(ctx, `
UPDATE report_schedules
SET report_key = $2, format = $3, destination = $4, owner_user_id = $5, cron_expr = $6, spec = $7, enabled = $8, next_run_at = $9, updated_at = NOW()
WHERE id = $1`,
		parsed, req.ReportKey, format, destination, ownerUUID, req.CronExpr, spec, enabled, nextRun,
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
SELECT id, customer_id, report_key, format, destination, owner_user_id, cron_expr, spec, enabled, next_run_at
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
			var ownerUserID *uuid.UUID
			if err := rows.Scan(&row.id, &row.customerID, &row.reportKey, &row.format, &row.destination, &ownerUserID, &row.cronExpr, &row.specJSON, &row.enabled, &row.nextRunAt); err != nil {
				return err
			}
			if ownerUserID != nil {
				row.ownerUserID = ownerUserID.String()
			}
			if row.destination == "" {
				row.destination = "download"
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
UPDATE report_schedules
SET last_job_id = $2, last_run_status = $3, last_run_error_public = NULL, updated_at = NOW()
WHERE id = $1`, sid, jid, JobStatusPending)
	return err
}

func markReportScheduleEnqueueFailed(ctx context.Context, pool *pgxpool.Pool, scheduleID, publicErr string) error {
	sid, err := uuid.Parse(scheduleID)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `
UPDATE report_schedules
SET last_run_status = $2, last_run_error_public = $3, updated_at = NOW()
WHERE id = $1`, sid, JobStatusFailed, publicErr)
	return err
}

func enrichReportScheduleFromLastJob(ctx context.Context, pool *pgxpool.Pool, dto *ReportScheduleDTO) {
	if dto == nil || dto.LastJobID == "" || pool == nil {
		return
	}
	var status string
	var errMsg *string
	parsed, err := uuid.Parse(dto.LastJobID)
	if err != nil {
		return
	}
	err = pool.QueryRow(ctx, `
SELECT status, error_message FROM report_jobs WHERE id = $1`, parsed).Scan(&status, &errMsg)
	if err != nil {
		return
	}
	dto.LastRunStatus = status
	if status == JobStatusFailed && errMsg != nil {
		dto.LastRunErrorPublic = SanitizeExportJobError(*errMsg)
	} else if status == JobStatusCompleted {
		dto.LastRunErrorPublic = ""
	}
}

func buildReportJobSpecFromSchedule(row reportScheduleRow) (ReportJobSpec, string, error) {
	rangeSpec := parseReportScheduleSpec(row.specJSON)
	now := time.Now().UTC()
	to := now.Add(time.Duration(rangeSpec.ToOffsetDays) * 24 * time.Hour)
	fromDays := rangeSpec.FromOffsetDays
	if fromDays <= 0 {
		fromDays = 7
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
	destination := row.destination
	if destination == "" {
		destination = "download"
	}
	jobSpec := ReportJobSpec{
		CustomerID:  row.customerID.String(),
		ReportKey:   row.reportKey,
		From:        from.Format(time.RFC3339),
		To:          to.Format(time.RFC3339),
		CompareFrom: strings.TrimSpace(rangeSpec.CompareFrom),
		CompareTo:   strings.TrimSpace(rangeSpec.CompareTo),
		Format:      format,
		Destination: destination,
		GoogleSheet: rangeSpec.GoogleSheet,
		Notify:      rangeSpec.Notify,
		ExportedBy:  strings.TrimSpace(row.ownerUserID),
		RowLimit:    rangeSpec.RowLimit,
	}
	if err := validateReportJobCompareSpec(jobSpec); err != nil {
		return ReportJobSpec{}, "", err
	}
	if err := normalizeReportJobNotify(&jobSpec); err != nil {
		return ReportJobSpec{}, "", err
	}
	idem := fmt.Sprintf("schedule:%s:%s", row.id.String(), row.nextRunAt.UTC().Format("2006-01-02T15:04"))
	return jobSpec, idem, nil
}

func (r *ReportJobRunner) validateScheduleSheetsOAuth(ctx context.Context, destination, ownerUserID string, enabled bool) error {
	if !enabled || normalizeReportScheduleDestination(destination) != "google_sheet" {
		return nil
	}
	ownerUserID = strings.TrimSpace(ownerUserID)
	if ownerUserID == "" {
		return fmt.Errorf("owner_user_id required for google_sheet schedules")
	}
	if r == nil || r.deps.ValidateGoogleSheetsOAuth == nil {
		return fmt.Errorf("google sheets export not configured")
	}
	return r.deps.ValidateGoogleSheetsOAuth(ctx, ownerUserID)
}

func (r *ReportJobRunner) RunReportScheduleNow(ctx context.Context, scheduleID string) (string, ReportScheduleDTO, error) {
	if r == nil || !r.pgEnabled() {
		return "", ReportScheduleDTO{}, fmt.Errorf("report schedule store unavailable")
	}
	row, err := loadReportScheduleRow(ctx, r.deps.Pool, scheduleID)
	if err != nil {
		return "", ReportScheduleDTO{}, err
	}
	spec, _, err := buildReportJobSpecFromSchedule(row)
	if err != nil {
		return "", ReportScheduleDTO{}, err
	}
	if err := r.validateScheduleSheetsOAuth(ctx, row.destination, row.ownerUserID, true); err != nil {
		return "", ReportScheduleDTO{}, err
	}
	idem := fmt.Sprintf("schedule-run-now:%s:%s", row.id.String(), time.Now().UTC().Format("2006-01-02T15:04"))
	jobID, err := r.CreateJob(ctx, spec, idem)
	if err != nil {
		_ = markReportScheduleEnqueueFailed(ctx, r.deps.Pool, scheduleID, SanitizeExportJobError(err.Error()))
		return "", ReportScheduleDTO{}, err
	}
	_ = markReportScheduleJob(ctx, r.deps.Pool, scheduleID, jobID)
	dto, err := getReportSchedule(ctx, r.deps.Pool, scheduleID)
	return jobID, dto, err
}

func loadReportScheduleRow(ctx context.Context, pool *pgxpool.Pool, id string) (reportScheduleRow, error) {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return reportScheduleRow{}, pgx.ErrNoRows
	}
	var row reportScheduleRow
	var ownerUserID *uuid.UUID
	err = pool.QueryRow(ctx, `
SELECT id, customer_id, report_key, format, destination, owner_user_id, cron_expr, spec, enabled, next_run_at
FROM report_schedules WHERE id = $1`, parsed).Scan(
		&row.id, &row.customerID, &row.reportKey, &row.format, &row.destination, &ownerUserID, &row.cronExpr, &row.specJSON, &row.enabled, &row.nextRunAt,
	)
	if err != nil {
		return reportScheduleRow{}, err
	}
	if ownerUserID != nil {
		row.ownerUserID = ownerUserID.String()
	}
	if row.destination == "" {
		row.destination = "download"
	}
	return row, nil
}
