package reportjob

import (
	"context"
	"encoding/json"
	"errors"
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
	permAny := h.RequireAnyPermission
	if permAny == nil {
		permAny = func(perms []string, next http.HandlerFunc) http.HandlerFunc {
			if len(perms) == 0 {
				return next
			}
			return perm(perms[0], next)
		}
	}
	readExports := []string{"exports:read", "customers:read", "campaigns:read"}
	runExports := []string{"exports:run", "customers:read", "campaigns:write"}

	mux.HandleFunc("POST /api/v1/reports/jobs", limit(permAny(runExports, h.postReportJob)))
	mux.HandleFunc("GET /api/v1/reports/jobs/{id}", limit(permAny(readExports, h.getReportJob)))
	mux.HandleFunc("GET /api/v1/reports/jobs/{id}/download", limit(permAny(readExports, h.downloadReportJob)))
	mux.HandleFunc("DELETE /api/v1/reports/jobs/{id}", limit(permAny(runExports, h.deleteReportJob)))
	mux.HandleFunc("POST /api/v1/reports/jobs/{id}/rerun", limit(permAny(runExports, h.rerunReportJob)))
	mux.HandleFunc("GET /api/v1/reports/notifications", limit(perm("exports:read", h.listReportExportNotifications)))
	mux.HandleFunc("POST /api/v1/reports/notifications/{id}/ack", limit(perm("exports:read", h.ackReportExportNotification)))
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
		msg := err.Error()
		if !isSafeExportValidationMessage(msg) {
			msg = SanitizeExportJobError(msg)
		}
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", msg)
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
		if errors.Is(err, ErrUseSpreadsheetURL) {
			httpresponse.Error(w, http.StatusConflict, "USE_SPREADSHEET_URL", "open spreadsheet_url from job status")
			return
		}
		httpresponse.Error(w, http.StatusConflict, "NOT_READY", SanitizeExportJobError(err.Error()))
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
	// Streams completed file from REPORT_EXPORT_DIR; no ClickHouse/Postgres read on download path.
	http.ServeContent(w, r, reportJobDownloadFilename(status), time.Now().UTC(), f)
}

func reportJobDownloadContentType(status ReportJobStatusDTO) string {
	switch {
	case status.Format == "zip" || status.ReportKey == "fraud-evidence-pack-bulk":
		return "application/zip"
	case status.Format == "json" || status.ReportKey == CampaignImportValidationReportKey:
		return "application/json"
	case status.Format == "xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
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
	case status.Format == "xlsx":
		ext = "xlsx"
	}
	return status.ReportKey + "." + ext
}

func (h *HTTPHandlers) rerunReportJob(w http.ResponseWriter, r *http.Request) {
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
	idemKey := r.Header.Get("Idempotency-Key")
	if idemKey == "" {
		idemKey = "rerun-" + jobID
	}
	newJobID, err := h.Runner.RerunJob(r.Context(), jobID, idemKey)
	if err != nil {
		msg := err.Error()
		if msg == "job not found" {
			httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", msg)
			return
		}
		if !isSafeExportValidationMessage(msg) {
			msg = SanitizeExportJobError(msg)
		}
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", msg)
		return
	}
	newStatus, _ := h.Runner.GetJob(r.Context(), newJobID)
	httpresponse.JSON(w, http.StatusCreated, newStatus)
}

func (h *HTTPHandlers) listReportExportNotifications(w http.ResponseWriter, r *http.Request) {
	userID := exportActorLabel(r.Context())
	if userID == "" {
		httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "operator session required")
		return
	}
	list, err := h.Runner.ListExportNotifications(r.Context(), userID, 20)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, list)
}

func (h *HTTPHandlers) ackReportExportNotification(w http.ResponseWriter, r *http.Request) {
	userID := exportActorLabel(r.Context())
	if userID == "" {
		httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "operator session required")
		return
	}
	notificationID := r.PathValue("id")
	dto, ok, err := h.Runner.AckExportNotification(r.Context(), userID, notificationID)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	if !ok {
		httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", "notification not found")
		return
	}
	httpresponse.JSON(w, http.StatusOK, dto)
}

func (h *HTTPHandlers) writeServiceError(w http.ResponseWriter, err error) {
	if h.WriteServiceError != nil {
		h.WriteServiceError(w, err)
		return
	}
	httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL", err.Error())
}
