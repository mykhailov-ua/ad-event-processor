package reportjob

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"

	"github.com/jackc/pgx/v5/pgxpool"
)

type HTTPHandlers struct {
	Runner                  *ReportJobRunner
	Pool                    *pgxpool.Pool
	ApplyRateLimit          func(http.HandlerFunc) http.HandlerFunc
	RequirePermission       func(string, http.HandlerFunc) http.HandlerFunc
	RequireAnyPermission    func([]string, http.HandlerFunc) http.HandlerFunc
	AuthorizeCustomerAccess func(*http.Request, string) error
	ValidateReportSchedule  func(context.Context, string, string, json.RawMessage) error
	WriteServiceError       func(http.ResponseWriter, error)
}

func (h *HTTPHandlers) Register(mux *http.ServeMux) {
	if h == nil || mux == nil {
		return
	}
	h.registerReportJobs(mux)
	h.registerReportSchedules(mux)
}

func (h *HTTPHandlers) registerReportJobs(mux *http.ServeMux) {
	if h.Runner == nil {
		return
	}
	limit := h.ApplyRateLimit
	if limit == nil {
		limit = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	perm := h.RequirePermission
	if perm == nil {
		perm = func(_ string, next http.HandlerFunc) http.HandlerFunc { return next }
	}
	mux.HandleFunc("POST /api/v1/reports/jobs", limit(perm("customers:read", h.postReportJob)))
	mux.HandleFunc("GET /api/v1/reports/jobs/{id}", limit(perm("customers:read", h.getReportJob)))
	mux.HandleFunc("GET /api/v1/reports/jobs/{id}/download", limit(perm("customers:read", h.downloadReportJob)))
	mux.HandleFunc("DELETE /api/v1/reports/jobs/{id}", limit(perm("customers:read", h.deleteReportJob)))
}

func (h *HTTPHandlers) postReportJob(w http.ResponseWriter, r *http.Request) {
	// 64KiB POST cap; export payload is written async to disk after job enqueue.
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
	if h.AuthorizeCustomerAccess != nil {
		if err := h.AuthorizeCustomerAccess(r, spec.CustomerID); err != nil {
			h.writeServiceError(w, err)
			return
		}
	}
	spec.RedactionProfile = resolveExportRedactionProfile(r.Context())
	spec.ExportedBy = exportActorLabel(r.Context())
	idemKey := r.Header.Get("Idempotency-Key") // PG unique key; duplicate POST returns original job id
	jobID, err := h.Runner.CreateJob(r.Context(), spec, idemKey)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	status, _ := h.Runner.GetJob(r.Context(), jobID)
	httpresponse.JSON(w, http.StatusCreated, status)
}

func (h *HTTPHandlers) getReportJob(w http.ResponseWriter, r *http.Request) {
	jobID := r.PathValue("id")
	status, ok := h.Runner.GetJob(r.Context(), jobID)
	if !ok {
		httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", "job not found")
		return
	}
	if h.AuthorizeCustomerAccess != nil {
		if err := h.AuthorizeCustomerAccess(r, status.CustomerID); err != nil {
			h.writeServiceError(w, err)
			return
		}
	}
	httpresponse.JSON(w, http.StatusOK, status)
}

func (h *HTTPHandlers) deleteReportJob(w http.ResponseWriter, r *http.Request) {
	jobID := r.PathValue("id")
	status, ok, err := h.Runner.CancelJob(r.Context(), jobID)
	if err != nil {
		httpresponse.Error(w, http.StatusConflict, "CONFLICT", err.Error())
		return
	}
	if !ok {
		httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", "job not found")
		return
	}
	if h.AuthorizeCustomerAccess != nil {
		if err := h.AuthorizeCustomerAccess(r, status.CustomerID); err != nil {
			h.writeServiceError(w, err)
			return
		}
	}
	httpresponse.JSON(w, http.StatusOK, status)
}

func (h *HTTPHandlers) downloadReportJob(w http.ResponseWriter, r *http.Request) {
	jobID := r.PathValue("id")
	f, status, err := h.Runner.OpenDownload(r.Context(), jobID)
	if err != nil {
		if status.ID == "" {
			httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", "job not found")
			return
		}
		httpresponse.Error(w, http.StatusConflict, "NOT_READY", err.Error())
		return
	}
	defer func() { _ = f.Close() }()
	if h.AuthorizeCustomerAccess != nil {
		if err := h.AuthorizeCustomerAccess(r, status.CustomerID); err != nil {
			h.writeServiceError(w, err)
			return
		}
	}
	w.Header().Set("Content-Type", reportJobDownloadContentType(status))
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, reportJobDownloadFilename(status)))
	// Streams completed file from REPORT_EXPORT_DIR; no CH/PG read on download path.
	http.ServeContent(w, r, reportJobDownloadFilename(status), time.Now().UTC(), f)
}

func reportJobDownloadContentType(status ReportJobStatusDTO) string {
	switch {
	case status.Format == "zip" || status.ReportKey == "fraud-evidence-pack-bulk":
		return "application/zip"
	case status.Format == "json" || status.ReportKey == CampaignImportValidationReportKey:
		return "application/json"
	default:
		return "text/csv"
	}
}

func reportJobDownloadFilename(status ReportJobStatusDTO) string {
	ext := "csv"
	switch {
	case status.Format == "zip" || status.ReportKey == "fraud-evidence-pack-bulk":
		ext = "zip"
	case status.Format == "json" || status.ReportKey == CampaignImportValidationReportKey:
		ext = "json"
	}
	return status.ReportKey + "." + ext
}

func (h *HTTPHandlers) writeServiceError(w http.ResponseWriter, err error) {
	if h.WriteServiceError != nil {
		h.WriteServiceError(w, err)
		return
	}
	httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL", err.Error())
}
