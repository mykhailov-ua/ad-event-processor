package reportjob

import (
	"errors"
	"net/http"

	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (h *HTTPHandlers) registerReportSchedules(mux *http.ServeMux) {
	if h == nil || h.Pool == nil {
		return
	}
	limit := h.ApplyRateLimit
	perm := h.RequirePermission
	permAny := h.RequireAnyPermission
	if permAny == nil {
		permAny = func(_ []string, next http.HandlerFunc) http.HandlerFunc { return next }
	}
	readPerms := []string{"campaigns:read", "campaigns:read:masked"}
	mux.HandleFunc("GET /api/v1/report-schedules", limit(permAny(readPerms, h.listReportSchedules)))
	mux.HandleFunc("GET /api/v1/report-schedules/{id}", limit(permAny(readPerms, h.getReportSchedule)))
	mux.HandleFunc("POST /api/v1/report-schedules", limit(perm("campaigns:write", h.createReportSchedule)))
	mux.HandleFunc("PUT /api/v1/report-schedules/{id}", limit(perm("campaigns:write", h.updateReportSchedule)))
	mux.HandleFunc("DELETE /api/v1/report-schedules/{id}", limit(perm("campaigns:write", h.deleteReportSchedule)))
}

func (h *HTTPHandlers) createReportSchedule(w http.ResponseWriter, r *http.Request) {
	// 64KiB body cap (coldpath.DefaultMaxBody); schedule spec JSON only, no export payload.
	req, err := coldpath.DecodeRequest[CreateReportScheduleRequest](w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}
	if _, err := uuid.Parse(req.CustomerID); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid customer_id")
		return
	}
	if req.ReportKey == "" || req.CronExpr == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "report_key and cron_expr required")
		return
	}
	if h.AuthorizeCustomerAccess != nil {
		if err := h.AuthorizeCustomerAccess(r, req.CustomerID); err != nil {
			h.writeServiceError(w, err)
			return
		}
	}
	if h.ValidateReportSchedule != nil {
		if err := h.ValidateReportSchedule(r.Context(), req.CustomerID, req.ReportKey, req.Spec); err != nil {
			if errors.Is(err, ErrForbidden) {
				httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", err.Error())
				return
			}
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			return
		}
	}
	created, err := insertReportSchedule(r.Context(), h.Pool, req)
	if err != nil {
		if errors.Is(err, errInvalidCronExpr) || err.Error() == "invalid cron_expr" {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			return
		}
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusCreated, created)
}

func (h *HTTPHandlers) listReportSchedules(w http.ResponseWriter, r *http.Request) {
	customerID := r.URL.Query().Get("customer_id")
	if customerID == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "customer_id query parameter is required")
		return
	}
	if _, err := uuid.Parse(customerID); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid customer_id")
		return
	}
	if h.AuthorizeCustomerAccess != nil {
		if err := h.AuthorizeCustomerAccess(r, customerID); err != nil {
			h.writeServiceError(w, err)
			return
		}
	}
	rows, err := listReportSchedules(r.Context(), h.Pool, customerID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, rows)
}

func (h *HTTPHandlers) getReportSchedule(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	row, err := getReportSchedule(r.Context(), h.Pool, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", "schedule not found")
			return
		}
		h.writeServiceError(w, err)
		return
	}
	if h.AuthorizeCustomerAccess != nil {
		if err := h.AuthorizeCustomerAccess(r, row.CustomerID); err != nil {
			h.writeServiceError(w, err)
			return
		}
	}
	httpresponse.JSON(w, http.StatusOK, row)
}

func (h *HTTPHandlers) updateReportSchedule(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	// Same 64KiB limit as create; full schedule replace, not partial patch.
	req, err := coldpath.DecodeRequest[UpdateReportScheduleRequest](w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}
	if req.ReportKey == "" || req.CronExpr == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "report_key and cron_expr required")
		return
	}
	existing, err := getReportSchedule(r.Context(), h.Pool, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", "schedule not found")
			return
		}
		h.writeServiceError(w, err)
		return
	}
	if h.AuthorizeCustomerAccess != nil {
		if err := h.AuthorizeCustomerAccess(r, existing.CustomerID); err != nil {
			h.writeServiceError(w, err)
			return
		}
	}
	if h.ValidateReportSchedule != nil {
		if err := h.ValidateReportSchedule(r.Context(), existing.CustomerID, req.ReportKey, req.Spec); err != nil {
			if errors.Is(err, ErrForbidden) {
				httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", err.Error())
				return
			}
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			return
		}
	}
	updated, err := updateReportSchedule(r.Context(), h.Pool, id, req)
	if err != nil {
		if errors.Is(err, errInvalidCronExpr) || err.Error() == "invalid cron_expr" {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			return
		}
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, updated)
}

func (h *HTTPHandlers) deleteReportSchedule(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	existing, err := getReportSchedule(r.Context(), h.Pool, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", "schedule not found")
			return
		}
		h.writeServiceError(w, err)
		return
	}
	if h.AuthorizeCustomerAccess != nil {
		if err := h.AuthorizeCustomerAccess(r, existing.CustomerID); err != nil {
			h.writeServiceError(w, err)
			return
		}
	}
	if err := deleteReportSchedule(r.Context(), h.Pool, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", "schedule not found")
			return
		}
		h.writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
