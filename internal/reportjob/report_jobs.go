package reportjob

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	// In-memory dev fallback caps; Postgres-backed runner uses report_jobs rows instead.
	reportJobMaxRecords = 512
	reportJobTTL        = 24 * time.Hour
)

type ReportJobGoogleSheetSpec struct {
	Mode          string `json:"mode,omitempty"`
	SpreadsheetID string `json:"spreadsheet_id,omitempty"`
	SheetTitle    string `json:"sheet_title,omitempty"`
}

type ReportJobSpec struct {
	CustomerID       string                   `json:"customer_id"`
	ReportKey        string                   `json:"report_key"`
	From             string                   `json:"from"`
	To               string                   `json:"to"`
	CompareFrom      string                   `json:"compare_from,omitempty"`
	CompareTo        string                   `json:"compare_to,omitempty"`
	Format           string                   `json:"format"`
	Destination      string                   `json:"destination,omitempty"`
	GoogleSheet      ReportJobGoogleSheetSpec `json:"google_sheet,omitempty"`
	Notify           ReportJobNotifySpec      `json:"notify,omitempty"`
	RedactionProfile string                   `json:"redaction_profile,omitempty"`
	ExportedBy       string                   `json:"exported_by,omitempty"`
	ImportSourceKind string                   `json:"import_source_kind,omitempty"`
	ImportPayload    json.RawMessage          `json:"import_payload,omitempty"`
	RowLimit         int                      `json:"row_limit,omitempty"`
}

type ReportJobStatusDTO struct {
	ID             string `json:"id"`
	JobID          string `json:"job_id"`
	CustomerID     string `json:"customer_id"`
	ReportKey      string `json:"report_key"`
	Format         string `json:"format"`
	Destination    string `json:"destination,omitempty"`
	Status         string `json:"status"`
	Bytes          int64  `json:"bytes,omitempty"`
	Error          string `json:"error,omitempty"`
	SpreadsheetURL string `json:"spreadsheet_url,omitempty"`
	SpreadsheetID  string `json:"spreadsheet_id,omitempty"`
	CreatedAt      string `json:"created_at"`
}

type GoogleSheetExportResult struct {
	SpreadsheetID  string
	SpreadsheetURL string
}

type reportJobRecord struct {
	spec           ReportJobSpec
	idempotencyKey string
	status         string
	bytes          int64
	errMsg         string
	filePath       string
	spreadsheetID  string
	spreadsheetURL string
	createdAt      time.Time
}

type ReportJobRunner struct {
	exportDir      string
	deps           ExportDeps
	mu             sync.RWMutex
	jobs           map[string]*reportJobRecord
	byIdem         map[string]string
	notifMu        sync.RWMutex
	notifs         []exportNotificationRecord
	notifByJobUser map[string]string
}

func NewReportJobRunner(exportDir string, deps ExportDeps) *ReportJobRunner {
	if exportDir == "" {
		exportDir = DefaultReportExportDirPath()
	}
	return &ReportJobRunner{
		exportDir:      exportDir,
		deps:           deps,
		jobs:           make(map[string]*reportJobRecord),
		byIdem:         make(map[string]string),
		notifByJobUser: make(map[string]string),
	}
}

func (r *ReportJobRunner) CreateJob(ctx context.Context, spec ReportJobSpec, idempotencyKey string) (string, error) {
	if _, err := uuid.Parse(spec.CustomerID); err != nil {
		return "", fmt.Errorf("invalid customer_id")
	}
	if spec.ReportKey == "" {
		return "", fmt.Errorf("report_key required")
	}
	format := spec.Format
	if format == "" {
		if spec.ReportKey == "fraud-evidence-pack-bulk" {
			format = "zip"
		} else {
			format = "csv"
		}
	}
	spec.Format = format
	if format != "csv" && format != "json" && format != "zip" && format != "xlsx" {
		return "", fmt.Errorf("format must be csv, json, zip, or xlsx")
	}
	if spec.ReportKey == "fraud-evidence-pack-bulk" && format != "zip" {
		return "", fmt.Errorf("format must be zip for fraud-evidence-pack-bulk")
	}
	destination := strings.TrimSpace(spec.Destination)
	if destination == "" {
		destination = "download"
	}
	spec.Destination = destination
	if destination == "google_sheet" {
		if format != "csv" && format != "xlsx" {
			return "", fmt.Errorf("format must be csv or xlsx for google_sheet destination")
		}
		mode := strings.TrimSpace(spec.GoogleSheet.Mode)
		if mode == "" {
			spec.GoogleSheet.Mode = "create"
		} else if mode != "create" && mode != "append" {
			return "", fmt.Errorf("google_sheet.mode must be create or append")
		}
		if spec.GoogleSheet.Mode == "append" && strings.TrimSpace(spec.GoogleSheet.SpreadsheetID) == "" {
			return "", fmt.Errorf("google_sheet.spreadsheet_id required for append mode")
		}
		operatorID := strings.TrimSpace(spec.ExportedBy)
		if operatorID == "" {
			return "", fmt.Errorf("google sheets export requires connected operator")
		}
		if r.deps.ValidateGoogleSheetsOAuth == nil {
			return "", fmt.Errorf("google sheets export not configured")
		}
		if err := r.deps.ValidateGoogleSheetsOAuth(ctx, operatorID); err != nil {
			return "", err
		}
	}
	spec.RowLimit = r.normalizeExportRowLimit(ctx, spec.ReportKey, spec.RowLimit)
	if _, _, err := ParseReportRangeFromStrings(spec.From, spec.To); err != nil {
		if spec.ReportKey != CampaignImportValidationReportKey {
			return "", err
		}
	}
	if err := validateReportJobCompareSpec(spec); err != nil {
		return "", err
	}
	if err := normalizeReportJobNotify(&spec); err != nil {
		return "", err
	}

	if r.pgEnabled() {
		return r.createJobPG(ctx, spec, idempotencyKey)
	}

	// In-memory queue: local idempotency map + eviction; export runs in goroutine (no SKIP LOCKED worker).
	r.mu.Lock()
	defer r.mu.Unlock()
	r.evictLocked(time.Now().UTC())

	if idempotencyKey != "" {
		if existing, ok := r.byIdem[idempotencyKey]; ok {
			return existing, nil
		}
	}
	if len(r.jobs) >= reportJobMaxRecords {
		return "", fmt.Errorf("report job queue full")
	}

	jobID := uuid.New().String()
	rec := &reportJobRecord{
		spec:           spec,
		idempotencyKey: idempotencyKey,
		status:         JobStatusPending,
		createdAt:      time.Now().UTC(),
	}
	r.jobs[jobID] = rec
	if idempotencyKey != "" {
		r.byIdem[idempotencyKey] = jobID
	}
	go r.runJob(ctx, jobID, spec)
	return jobID, nil
}

func (r *ReportJobRunner) GetJob(ctx context.Context, jobID string) (ReportJobStatusDTO, bool) {
	if r.pgEnabled() {
		dto, ok, err := r.getJobPG(ctx, jobID)
		if err != nil {
			return ReportJobStatusDTO{}, false
		}
		return dto, ok
	}
	r.mu.RLock()
	rec, ok := r.jobs[jobID]
	r.mu.RUnlock()
	if !ok {
		return ReportJobStatusDTO{}, false
	}
	return r.toDTO(jobID, rec), true
}

func (r *ReportJobRunner) CancelJob(ctx context.Context, jobID string) (ReportJobStatusDTO, bool, error) {
	if r.pgEnabled() {
		dto, ok, err := r.cancelJobPG(ctx, jobID)
		return dto, ok, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	rec, ok := r.jobs[jobID]
	if !ok {
		return ReportJobStatusDTO{}, false, nil
	}
	if rec.status != JobStatusPending {
		return r.toDTO(jobID, rec), false, fmt.Errorf("job cannot be cancelled")
	}
	rec.status = JobStatusCancelled
	return r.toDTO(jobID, rec), true, nil
}

func (r *ReportJobRunner) ListJobsByCustomer(ctx context.Context, customerID string, limit int) []ReportJobStatusDTO {
	if limit <= 0 {
		limit = 10
	}
	if r.pgEnabled() {
		out, err := r.listJobsByCustomerPG(ctx, customerID, limit)
		if err != nil {
			return nil
		}
		return out
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	type item struct {
		id  string
		rec *reportJobRecord
	}
	items := make([]item, 0, len(r.jobs))
	for id, rec := range r.jobs {
		if rec.spec.CustomerID == customerID {
			items = append(items, item{id: id, rec: rec})
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].rec.createdAt.After(items[j].rec.createdAt)
	})
	if len(items) > limit {
		items = items[:limit]
	}
	out := make([]ReportJobStatusDTO, 0, len(items))
	for _, it := range items {
		out = append(out, r.toDTO(it.id, it.rec))
	}
	return out
}

func (r *ReportJobRunner) OpenDownload(ctx context.Context, jobID string) (*os.File, ReportJobStatusDTO, error) {
	if r.pgEnabled() {
		path, dto, err := r.openDownloadPG(ctx, jobID)
		if err != nil {
			return nil, dto, err
		}
		f, err := os.Open(path)
		if err != nil {
			return nil, dto, err
		}
		return f, dto, nil
	}
	r.mu.RLock()
	rec, ok := r.jobs[jobID]
	r.mu.RUnlock()
	if !ok {
		return nil, ReportJobStatusDTO{}, fmt.Errorf("job not found")
	}
	dto := r.toDTO(jobID, rec)
	if rec.spec.Destination == "google_sheet" || rec.spreadsheetURL != "" {
		return nil, dto, ErrUseSpreadsheetURL
	}
	if rec.status != JobStatusCompleted || rec.filePath == "" {
		return nil, dto, fmt.Errorf("export not ready")
	}
	f, err := os.Open(rec.filePath)
	if err != nil {
		return nil, dto, err
	}
	return f, dto, nil
}

func (r *ReportJobRunner) normalizeExportRowLimit(ctx context.Context, reportKey string, requested int) int {
	chunkBytes := uint64(0)
	if r != nil && r.deps.ExportChunkMaxBytes != nil {
		chunkBytes = uint64(r.deps.ExportChunkMaxBytes(ctx))
	}
	tierMax := ResolveExportRowLimitTierMax(chunkBytes, r.reportLicenseGated(reportKey))
	return NormalizeExportRowLimit(requested, tierMax)
}

func (r *ReportJobRunner) reportLicenseGated(reportKey string) bool {
	if r != nil && r.deps.ReportLicenseGated != nil {
		return r.deps.ReportLicenseGated(reportKey)
	}
	return false
}

func (r *ReportJobRunner) toDTO(jobID string, rec *reportJobRecord) ReportJobStatusDTO {
	destination := rec.spec.Destination
	if destination == "" {
		destination = "download"
	}
	return ReportJobStatusDTO{
		ID:             jobID,
		JobID:          jobID,
		CustomerID:     rec.spec.CustomerID,
		ReportKey:      rec.spec.ReportKey,
		Format:         rec.spec.Format,
		Destination:    destination,
		Status:         rec.status,
		Bytes:          rec.bytes,
		Error:          SanitizeExportJobError(rec.errMsg),
		SpreadsheetURL: rec.spreadsheetURL,
		SpreadsheetID:  rec.spreadsheetID,
		CreatedAt:      rec.createdAt.Format(time.RFC3339),
	}
}

func (r *ReportJobRunner) evictLocked(now time.Time) {
	// TTL eviction drops in-memory rows and deletes export files on local disk (OS boundary).
	for id, rec := range r.jobs {
		if now.Sub(rec.createdAt) <= reportJobTTL {
			continue
		}
		if rec.filePath != "" {
			_ = os.Remove(rec.filePath)
		}
		if rec.idempotencyKey != "" {
			delete(r.byIdem, rec.idempotencyKey)
		}
		delete(r.jobs, id)
	}
}

func (r *ReportJobRunner) runJob(parent context.Context, jobID string, spec ReportJobSpec) {
	jobCtx, cancel := context.WithTimeout(parent, reportJobRunTimeoutForSpec(spec))
	defer cancel()

	if !r.pgEnabled() {
		r.mu.Lock()
		rec, ok := r.jobs[jobID]
		if !ok {
			r.mu.Unlock()
			return
		}
		rec.status = JobStatusRunning
		spec = rec.spec
		r.mu.Unlock()
	}

	exportStart := time.Now()
	if spec.Destination == "google_sheet" {
		if r.deps.WriteGoogleSheet == nil {
			r.failJob(jobCtx, jobID, spec, fmt.Errorf("google sheets export not configured"))
			return
		}
		result, exportErr := r.deps.WriteGoogleSheet(jobCtx, spec)
		if exportErr != nil {
			observeReportQuery(spec.ReportKey, exportStart, exportErr)
			r.failJob(jobCtx, jobID, spec, exportErr)
			return
		}
		observeReportQuery(spec.ReportKey, exportStart, nil)
		if r.pgEnabled() {
			if err := completeReportJobGoogleSheetPG(jobCtx, r.deps.Pool, jobID, result.SpreadsheetID, result.SpreadsheetURL); err != nil {
				r.failJob(jobCtx, jobID, spec, err)
			} else {
				r.notifyJobTerminal(jobCtx, jobID, spec, JobStatusCompleted, "")
			}
			return
		}
		r.mu.Lock()
		if rec, ok := r.jobs[jobID]; ok {
			rec.status = JobStatusCompleted
			rec.spreadsheetID = result.SpreadsheetID
			rec.spreadsheetURL = result.SpreadsheetURL
		}
		r.mu.Unlock()
		r.notifyJobTerminal(jobCtx, jobID, spec, JobStatusCompleted, "")
		return
	}

	if err := os.MkdirAll(r.exportDir, 0o750); err != nil {
		r.failJob(jobCtx, jobID, spec, err)
		return
	}
	path := filepath.Join(r.exportDir, jobID+"."+ReportJobArtifactExt(spec))
	var exportErr error
	switch spec.ReportKey {
	case CampaignImportValidationReportKey:
		if r.deps.WriteCampaignImportValidation == nil {
			exportErr = fmt.Errorf("campaign import validation export not configured")
		} else {
			exportErr = r.deps.WriteCampaignImportValidation(jobCtx, path, spec)
		}
	default:
		if r.deps.WriteReport == nil {
			exportErr = fmt.Errorf("report export not configured")
		} else {
			exportErr = r.deps.WriteReport(jobCtx, path, spec)
		}
	}
	if exportErr != nil {
		observeReportQuery(spec.ReportKey, exportStart, exportErr)
		r.failJob(jobCtx, jobID, spec, exportErr)
		return
	}
	observeReportQuery(spec.ReportKey, exportStart, nil)
	info, err := os.Stat(path)
	if err != nil {
		r.failJob(jobCtx, jobID, spec, err)
		return
	}
	if r.pgEnabled() {
		if err := completeReportJobPG(jobCtx, r.deps.Pool, jobID, path, info.Size()); err != nil {
			r.failJob(jobCtx, jobID, spec, err)
		} else {
			r.notifyJobTerminal(jobCtx, jobID, spec, JobStatusCompleted, "")
		}
		return
	}
	r.mu.Lock()
	if rec, ok := r.jobs[jobID]; ok {
		rec.status = JobStatusCompleted
		rec.filePath = path
		rec.bytes = info.Size()
	}
	r.mu.Unlock()
	r.notifyJobTerminal(jobCtx, jobID, spec, JobStatusCompleted, "")
}

func (r *ReportJobRunner) failJob(ctx context.Context, jobID string, spec ReportJobSpec, err error) {
	publicErr := SanitizeExportJobErrorFromErr(err)
	if r.pgEnabled() {
		_ = failReportJobPG(ctx, r.deps.Pool, jobID, err.Error())
		r.notifyJobTerminal(ctx, jobID, spec, JobStatusFailed, publicErr)
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if rec, ok := r.jobs[jobID]; ok {
		rec.status = JobStatusFailed
		rec.errMsg = err.Error()
	}
	r.notifyJobTerminal(ctx, jobID, spec, JobStatusFailed, publicErr)
}

func ReportJobArtifactExt(spec ReportJobSpec) string {
	if spec.ReportKey == CampaignImportValidationReportKey {
		return "json"
	}
	if spec.ReportKey == "fraud-evidence-pack-bulk" {
		return "zip"
	}
	switch spec.Format {
	case "json":
		return "json"
	case "xlsx":
		return "xlsx"
	case "zip":
		return "zip"
	default:
		return "csv"
	}
}

func ParseReportRangeFromStrings(fromStr, toStr string) (time.Time, time.Time, error) {
	now := time.Now().UTC()
	to := now
	from := now.Add(-defaultReportLookback)
	if toStr != "" {
		parsed, err := time.Parse(time.RFC3339, toStr)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid to timestamp")
		}
		to = parsed.UTC()
	}
	if fromStr != "" {
		parsed, err := time.Parse(time.RFC3339, fromStr)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid from timestamp")
		}
		from = parsed.UTC()
	}
	if !from.Before(to) {
		return time.Time{}, time.Time{}, fmt.Errorf("from must be before to")
	}
	if to.Sub(from) > maxStatsRange {
		return time.Time{}, time.Time{}, fmt.Errorf("range exceeds 90 days")
	}
	return from, to, nil
}
