package adminapi

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/bidshard/ad-event-processor/pkg/coldpath"
	"github.com/bidshard/ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
)

const (
	JobStatusPending   = "PENDING"
	JobStatusRunning   = "RUNNING"
	JobStatusCompleted = "COMPLETED"
	JobStatusFailed    = "FAILED"

	defaultExportFetchRows  = 1000
	defaultExportJobTimeout = 15 * time.Minute
)

type JobSpec struct {
	CustomerID string `json:"customer_id"`
	From       string `json:"from"`
	To         string `json:"to"`
	Format     string `json:"format"`
}

type JobStatusDTO struct {
	ID          string `json:"id"`
	CustomerID  string `json:"customer_id"`
	Format      string `json:"format"`
	Status      string `json:"status"`
	Bytes       int64  `json:"bytes,omitempty"`
	DownloadURL string `json:"download_url,omitempty"`
	Error       string `json:"error,omitempty"`
	CreatedAt   string `json:"created_at"`
	CompletedAt string `json:"completed_at,omitempty"`
}

type jobRecord struct {
	spec        JobSpec
	customerID  uuid.UUID
	from        time.Time
	to          time.Time
	status      string
	bytes       int64
	errMsg      string
	filePath    string
	createdAt   time.Time
	completedAt time.Time
}

type ledgerExportReader interface {
	ListLedgerLinesInWindow(ctx context.Context, customerID uuid.UUID, from, to time.Time, cursorID int64, limit int32) ([]LedgerLineDTO, string, error)
}

type JobRunner struct {
	ledgerReads ledgerExportReader
	exportDir   string
	fetchRows   int32
	jobTimeout  time.Duration
	mu          sync.RWMutex
	jobs        map[string]*jobRecord
}

func NewJobRunner(ledgerReads ledgerExportReader, exportDir string) *JobRunner {
	if exportDir == "" {
		exportDir = "./data/billing-export"
	}
	return &JobRunner{
		ledgerReads: ledgerReads,
		exportDir:   exportDir,
		fetchRows:   defaultExportFetchRows,
		jobTimeout:  defaultExportJobTimeout,
		jobs:        make(map[string]*jobRecord),
	}
}

func (s *JobRunner) ConfigureExport(fetchRows int32, jobTimeout time.Duration) {
	if s == nil {
		return
	}
	if fetchRows > 0 {
		s.fetchRows = fetchRows
	}
	if jobTimeout > 0 {
		s.jobTimeout = jobTimeout
	}
}

func (s *JobRunner) CreateJob(ctx context.Context, spec JobSpec) (string, error) {
	if s == nil || s.ledgerReads == nil {
		return "", fmt.Errorf("export job runner not configured")
	}
	customerID, err := uuid.Parse(spec.CustomerID)
	if err != nil {
		return "", fmt.Errorf("invalid customer_id")
	}
	from, to, err := ParseStatementPeriod(spec.From, spec.To, "")
	if err != nil {
		return "", err
	}
	format := spec.Format
	if format == "" {
		format = "csv"
	}
	if format != "csv" && format != "ndjson" {
		return "", fmt.Errorf("format must be csv or ndjson")
	}

	jobID := uuid.New().String()
	rec := &jobRecord{
		spec:       spec,
		customerID: customerID,
		from:       from,
		to:         to,
		status:     JobStatusPending,
		createdAt:  time.Now().UTC(),
	}
	s.mu.Lock()
	s.jobs[jobID] = rec
	s.mu.Unlock()

	go func() {
		timeout := s.jobTimeout
		if timeout <= 0 {
			timeout = defaultExportJobTimeout
		}
		jobCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		s.runJob(jobCtx, jobID)
	}()
	return jobID, nil
}

func (s *JobRunner) GetJob(jobID string) (JobStatusDTO, bool) {
	s.mu.RLock()
	rec, ok := s.jobs[jobID]
	s.mu.RUnlock()
	if !ok {
		return JobStatusDTO{}, false
	}
	return s.toDTO(jobID, rec), true
}

func (s *JobRunner) OpenDownload(jobID string) (*os.File, JobStatusDTO, error) {
	s.mu.RLock()
	rec, ok := s.jobs[jobID]
	s.mu.RUnlock()
	if !ok {
		return nil, JobStatusDTO{}, fmt.Errorf("job not found")
	}
	if rec.status != JobStatusCompleted || rec.filePath == "" {
		return nil, s.toDTO(jobID, rec), fmt.Errorf("export not ready")
	}
	f, err := os.Open(rec.filePath)
	if err != nil {
		return nil, JobStatusDTO{}, err
	}
	return f, s.toDTO(jobID, rec), nil
}

func (s *JobRunner) toDTO(jobID string, rec *jobRecord) JobStatusDTO {
	dto := JobStatusDTO{
		ID:         jobID,
		CustomerID: rec.customerID.String(),
		Format:     rec.spec.Format,
		Status:     rec.status,
		Bytes:      rec.bytes,
		CreatedAt:  rec.createdAt.UTC().Format(time.RFC3339),
	}
	if rec.status == JobStatusCompleted {
		dto.DownloadURL = "/api/v1/billing/exports/" + jobID + "/download"
	}
	if rec.errMsg != "" {
		dto.Error = rec.errMsg
	}
	if !rec.completedAt.IsZero() {
		dto.CompletedAt = rec.completedAt.UTC().Format(time.RFC3339)
	}
	return dto
}

func (s *JobRunner) runJob(ctx context.Context, jobID string) {
	s.mu.Lock()
	rec, ok := s.jobs[jobID]
	if !ok {
		s.mu.Unlock()
		return
	}
	rec.status = JobStatusRunning
	s.mu.Unlock()

	if err := os.MkdirAll(s.exportDir, 0o755); err != nil {
		s.failJob(jobID, err)
		return
	}

	ext := rec.spec.Format
	if ext == "" {
		ext = "csv"
	}
	filePath := filepath.Join(s.exportDir, jobID+"."+ext)
	var writeErr error
	switch ext {
	case "ndjson":
		writeErr = s.writeNDJSON(ctx, filePath, rec)
	default:
		writeErr = s.writeCSV(ctx, filePath, rec)
	}
	if writeErr != nil {
		s.failJob(jobID, writeErr)
		return
	}
	if err := ctx.Err(); err != nil {
		s.failJob(jobID, err)
		return
	}

	info, err := os.Stat(filePath)
	if err != nil {
		s.failJob(jobID, err)
		return
	}

	s.mu.Lock()
	if rec, ok := s.jobs[jobID]; ok {
		rec.status = JobStatusCompleted
		rec.filePath = filePath
		rec.bytes = info.Size()
		rec.completedAt = time.Now().UTC()
	}
	s.mu.Unlock()
}

func (s *JobRunner) failJob(jobID string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if rec, ok := s.jobs[jobID]; ok {
		rec.status = JobStatusFailed
		rec.errMsg = err.Error()
		rec.completedAt = time.Now().UTC()
	}
}

func (s *JobRunner) writeCSV(ctx context.Context, path string, rec *jobRecord) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	cw := csv.NewWriter(f)
	if err := cw.Write([]string{"id", "amount_micro", "ledger_type", "created_at"}); err != nil {
		return err
	}

	var cursor int64
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		lines, next, err := s.ledgerReads.ListLedgerLinesInWindow(ctx, rec.customerID, rec.from, rec.to, cursor, s.fetchRows)
		if err != nil {
			return err
		}
		for _, line := range lines {
			if err := cw.Write([]string{
				fmt.Sprintf("%d", line.ID),
				fmt.Sprintf("%d", line.AmountMicro),
				line.LedgerType,
				line.CreatedAt,
			}); err != nil {
				return err
			}
		}
		if next == "" {
			break
		}
		var parsed int64
		_, _ = fmt.Sscanf(next, "%d", &parsed)
		cursor = parsed
	}
	cw.Flush()
	return cw.Error()
}

func (s *JobRunner) writeNDJSON(ctx context.Context, path string, rec *jobRecord) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	enc := json.NewEncoder(f)
	var cursor int64
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		lines, next, err := s.ledgerReads.ListLedgerLinesInWindow(ctx, rec.customerID, rec.from, rec.to, cursor, s.fetchRows)
		if err != nil {
			return err
		}
		for _, line := range lines {
			if err := enc.Encode(line); err != nil {
				return err
			}
		}
		if next == "" {
			break
		}
		var parsed int64
		_, _ = fmt.Sscanf(next, "%d", &parsed)
		cursor = parsed
	}
	return nil
}

type ExportHTTPHandlers struct {
	JobRunner               *JobRunner
	ApplyRateLimit          func(http.HandlerFunc) http.HandlerFunc
	RequirePermission       func(string, http.HandlerFunc) http.HandlerFunc
	AuthorizeCustomerAccess func(*http.Request, string) error
	WriteServiceError       func(http.ResponseWriter, error)
}

func (exportHandlers *ExportHTTPHandlers) Register(mux *http.ServeMux) {
	if exportHandlers == nil || exportHandlers.JobRunner == nil {
		return
	}
	limit := exportHandlers.ApplyRateLimit
	perm := exportHandlers.RequirePermission
	if limit == nil {
		limit = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	if perm == nil {
		perm = func(_ string, next http.HandlerFunc) http.HandlerFunc { return next }
	}
	mux.HandleFunc("POST /api/v1/billing/exports", limit(perm("customers:read", exportHandlers.createExport)))
	mux.HandleFunc("GET /api/v1/billing/exports/{job_id}", limit(perm("customers:read", exportHandlers.getExport)))
	mux.HandleFunc("GET /api/v1/billing/exports/{job_id}/download", limit(perm("customers:read", exportHandlers.downloadExport)))
}

func (exportHandlers *ExportHTTPHandlers) createExport(w http.ResponseWriter, r *http.Request) {
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}
	spec, err := coldpath.DecodeBody[JobSpec](body)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if spec.CustomerID == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "customer_id is required")
		return
	}
	if exportHandlers.AuthorizeCustomerAccess != nil {
		if err := exportHandlers.AuthorizeCustomerAccess(r, spec.CustomerID); err != nil {
			exportHandlers.writeServiceError(w, err)
			return
		}
	}
	jobID, err := exportHandlers.JobRunner.CreateJob(r.Context(), spec)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	w.Header().Set("Location", "/api/v1/billing/exports/"+jobID)
	httpresponse.JSON(w, http.StatusAccepted, ExportJobCreatedResponse{JobID: jobID})
}

func (exportHandlers *ExportHTTPHandlers) getExport(w http.ResponseWriter, r *http.Request) {
	jobID := r.PathValue("job_id")
	status, ok := exportHandlers.JobRunner.GetJob(jobID)
	if !ok {
		httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", "export job not found")
		return
	}
	if exportHandlers.AuthorizeCustomerAccess != nil {
		if err := exportHandlers.AuthorizeCustomerAccess(r, status.CustomerID); err != nil {
			exportHandlers.writeServiceError(w, err)
			return
		}
	}
	httpresponse.JSON(w, http.StatusOK, status)
}

func (exportHandlers *ExportHTTPHandlers) downloadExport(w http.ResponseWriter, r *http.Request) {
	jobID := r.PathValue("job_id")
	f, status, err := exportHandlers.JobRunner.OpenDownload(jobID)
	if err != nil {
		if status.ID != "" {
			if exportHandlers.AuthorizeCustomerAccess != nil {
				if aerr := exportHandlers.AuthorizeCustomerAccess(r, status.CustomerID); aerr != nil {
					exportHandlers.writeServiceError(w, aerr)
					return
				}
			}
			httpresponse.Error(w, http.StatusConflict, "NOT_READY", err.Error())
			return
		}
		httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", "export job not found")
		return
	}
	defer func() { _ = f.Close() }()
	if exportHandlers.AuthorizeCustomerAccess != nil {
		if err := exportHandlers.AuthorizeCustomerAccess(r, status.CustomerID); err != nil {
			exportHandlers.writeServiceError(w, err)
			return
		}
	}
	if status.Format == "ndjson" {
		w.Header().Set("Content-Type", "application/x-ndjson")
	} else {
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	}
	w.Header().Set("Content-Disposition", "attachment; filename=\"billing-export-"+jobID+"."+status.Format+"\"")
	maxBytes := status.Bytes
	if maxBytes <= 0 {
		maxBytes = 256 << 20
	}
	if _, err := io.Copy(w, io.LimitReader(f, maxBytes)); err != nil {
		slog.Warn("export download interrupted", "job_id", jobID, "error", err)
	}
}

func (exportHandlers *ExportHTTPHandlers) writeServiceError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrForbidden) {
		httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "forbidden")
		return
	}
	if exportHandlers.WriteServiceError != nil {
		exportHandlers.WriteServiceError(w, err)
		return
	}
	httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL", "request failed")
}
