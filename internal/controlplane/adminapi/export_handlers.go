package adminapi

import (
	"errors"
	"espx/pkg/coldpath"
	"io"
	"net/http"

	"espx/pkg/httpresponse"
)

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
	httpresponse.JSON(w, http.StatusAccepted, map[string]any{"job_id": jobID})
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
	defer f.Close()
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
	_, _ = io.Copy(w, f)
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
