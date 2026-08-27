package controlplane

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
)

const (
	reportJobMaxRecords = 512
	reportJobTTL        = 24 * time.Hour
	reportJobRunTimeout = 2 * time.Minute
)

type ReportJobSpec struct {
	CustomerID string `json:"customer_id"`
	ReportKey  string `json:"report_key"`
	From       string `json:"from"`
	To         string `json:"to"`
	Format     string `json:"format"`
}

type ReportJobStatusDTO struct {
	ID         string `json:"id"`
	JobID      string `json:"job_id"`
	CustomerID string `json:"customer_id"`
	ReportKey  string `json:"report_key"`
	Format     string `json:"format"`
	Status     string `json:"status"`
	Bytes      int64  `json:"bytes,omitempty"`
	Error      string `json:"error,omitempty"`
	CreatedAt  string `json:"created_at"`
}

type reportJobRecord struct {
	spec           ReportJobSpec
	idempotencyKey string
	status         string
	bytes          int64
	errMsg         string
	filePath       string
	createdAt      time.Time
}

type ReportJobRunner struct {
	exportDir string
	deps      ReportExportDeps
	mu        sync.RWMutex
	jobs      map[string]*reportJobRecord
	byIdem    map[string]string
}

func NewReportJobRunner(exportDir string, deps ReportExportDeps) *ReportJobRunner {
	if exportDir == "" {
		exportDir = defaultReportExportDirPath()
	}
	return &ReportJobRunner{
		exportDir: exportDir,
		deps:      deps,
		jobs:      make(map[string]*reportJobRecord),
		byIdem:    make(map[string]string),
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
		format = "csv"
	}
	spec.Format = format
	if _, _, err := parseReportRangeFromStrings(spec.From, spec.To); err != nil {
		return "", err
	}

	if r.pgEnabled() {
		return r.createJobPG(ctx, spec, idempotencyKey)
	}

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
	if rec.status != JobStatusCompleted || rec.filePath == "" {
		return nil, dto, fmt.Errorf("export not ready")
	}
	f, err := os.Open(rec.filePath)
	if err != nil {
		return nil, dto, err
	}
	return f, dto, nil
}

func (r *ReportJobRunner) toDTO(jobID string, rec *reportJobRecord) ReportJobStatusDTO {
	return ReportJobStatusDTO{
		ID:         jobID,
		JobID:      jobID,
		CustomerID: rec.spec.CustomerID,
		ReportKey:  rec.spec.ReportKey,
		Format:     rec.spec.Format,
		Status:     rec.status,
		Bytes:      rec.bytes,
		Error:      rec.errMsg,
		CreatedAt:  rec.createdAt.Format(time.RFC3339),
	}
}

func (r *ReportJobRunner) evictLocked(now time.Time) {
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
	jobCtx, cancel := context.WithTimeout(parent, reportJobRunTimeout)
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

	if err := os.MkdirAll(r.exportDir, 0o750); err != nil {
		r.failJob(jobCtx, jobID, err)
		return
	}
	path := filepath.Join(r.exportDir, jobID+".csv")
	exportStart := time.Now()
	if err := r.writeReportCSV(jobCtx, path, spec); err != nil {
		observeReportQuery(spec.ReportKey, exportStart, err)
		r.failJob(jobCtx, jobID, err)
		return
	}
	observeReportQuery(spec.ReportKey, exportStart, nil)
	info, err := os.Stat(path)
	if err != nil {
		r.failJob(jobCtx, jobID, err)
		return
	}
	if r.pgEnabled() {
		if err := completeReportJobPG(jobCtx, r.deps.Pool, jobID, path, info.Size()); err != nil {
			r.failJob(jobCtx, jobID, err)
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
}

func (r *ReportJobRunner) failJob(ctx context.Context, jobID string, err error) {
	if r.pgEnabled() {
		_ = failReportJobPG(ctx, r.deps.Pool, jobID, err.Error())
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if rec, ok := r.jobs[jobID]; ok {
		rec.status = JobStatusFailed
		rec.errMsg = err.Error()
	}
}

func parseReportRangeFromStrings(fromStr, toStr string) (time.Time, time.Time, error) {
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

func (reports *ReportsHTTPHandlers) registerReportJobs(mux *http.ServeMux) {
	if reports.ReportJobs == nil {
		return
	}
	limit := reports.ApplyRateLimit
	perm := reports.RequirePermission
	mux.HandleFunc("POST /api/v1/reports/jobs", limit(perm("customers:read", reports.postReportJob)))
	mux.HandleFunc("GET /api/v1/reports/jobs/{id}", limit(perm("customers:read", reports.getReportJob)))
	mux.HandleFunc("GET /api/v1/reports/jobs/{id}/download", limit(perm("customers:read", reports.downloadReportJob)))
}

func (reports *ReportsHTTPHandlers) postReportJob(w http.ResponseWriter, r *http.Request) {
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "failed to read body")
		return
	}
	spec, err := coldpath.DecodeBody[ReportJobSpec](body)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid body")
		return
	}
	if reports.AuthorizeCustomerAccess != nil {
		if err := reports.AuthorizeCustomerAccess(r, spec.CustomerID); err != nil {
			reports.writeServiceError(w, err)
			return
		}
	}
	idemKey := r.Header.Get("Idempotency-Key")
	jobID, err := reports.ReportJobs.CreateJob(r.Context(), spec, idemKey)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	status, _ := reports.ReportJobs.GetJob(r.Context(), jobID)
	httpresponse.JSON(w, http.StatusCreated, status)
}

func (reports *ReportsHTTPHandlers) getReportJob(w http.ResponseWriter, r *http.Request) {
	jobID := r.PathValue("id")
	status, ok := reports.ReportJobs.GetJob(r.Context(), jobID)
	if !ok {
		httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", "job not found")
		return
	}
	if reports.AuthorizeCustomerAccess != nil {
		if err := reports.AuthorizeCustomerAccess(r, status.CustomerID); err != nil {
			reports.writeServiceError(w, err)
			return
		}
	}
	httpresponse.JSON(w, http.StatusOK, status)
}

func (reports *ReportsHTTPHandlers) downloadReportJob(w http.ResponseWriter, r *http.Request) {
	jobID := r.PathValue("id")
	f, status, err := reports.ReportJobs.OpenDownload(r.Context(), jobID)
	if err != nil {
		if status.ID == "" {
			httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", "job not found")
			return
		}
		httpresponse.Error(w, http.StatusConflict, "NOT_READY", err.Error())
		return
	}
	defer func() { _ = f.Close() }()
	if reports.AuthorizeCustomerAccess != nil {
		if err := reports.AuthorizeCustomerAccess(r, status.CustomerID); err != nil {
			reports.writeServiceError(w, err)
			return
		}
	}
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.csv"`, status.ReportKey))
	http.ServeContent(w, r, status.ReportKey+".csv", time.Now().UTC(), f)
}
