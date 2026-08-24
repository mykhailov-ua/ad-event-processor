package controlplane

import (
	"errors"
	"net/http"

	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (reports *ReportsHTTPHandlers) registerReportSchedules(mux *http.ServeMux) {
	if reports == nil || reports.Pool == nil {
		return
	}
	limit := reports.ApplyRateLimit
	perm := reports.RequirePermission
	permAny := reports.RequireAnyPermission
	if permAny == nil {
		permAny = func(_ []string, next http.HandlerFunc) http.HandlerFunc { return next }
	}
	readPerms := []string{"campaigns:read", "campaigns:read:masked"}
	mux.HandleFunc("GET /api/v1/report-schedules", limit(permAny(readPerms, reports.listReportSchedules)))
	mux.HandleFunc("GET /api/v1/report-schedules/{id}", limit(permAny(readPerms, reports.getReportSchedule)))
	mux.HandleFunc("POST /api/v1/report-schedules", limit(perm("campaigns:write", reports.createReportSchedule)))
	mux.HandleFunc("PUT /api/v1/report-schedules/{id}", limit(perm("campaigns:write", reports.updateReportSchedule)))
	mux.HandleFunc("DELETE /api/v1/report-schedules/{id}", limit(perm("campaigns:write", reports.deleteReportSchedule)))
}

func (reports *ReportsHTTPHandlers) createReportSchedule(w http.ResponseWriter, r *http.Request) {
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
	if reports.AuthorizeCustomerAccess != nil {
		if err := reports.AuthorizeCustomerAccess(r, req.CustomerID); err != nil {
			reports.writeServiceError(w, err)
			return
		}
	}
	created, err := insertReportSchedule(r.Context(), reports.Pool, req)
	if err != nil {
		if errors.Is(err, errInvalidCronExpr) || err.Error() == "invalid cron_expr" {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			return
		}
		reports.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusCreated, created)
}

func (reports *ReportsHTTPHandlers) listReportSchedules(w http.ResponseWriter, r *http.Request) {
	customerID := r.URL.Query().Get("customer_id")
	if customerID == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "customer_id query parameter is required")
		return
	}
	if _, err := uuid.Parse(customerID); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid customer_id")
		return
	}
	if reports.AuthorizeCustomerAccess != nil {
		if err := reports.AuthorizeCustomerAccess(r, customerID); err != nil {
			reports.writeServiceError(w, err)
			return
		}
	}
	rows, err := listReportSchedules(r.Context(), reports.Pool, customerID)
	if err != nil {
		reports.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, rows)
}

func (reports *ReportsHTTPHandlers) getReportSchedule(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	row, err := getReportSchedule(r.Context(), reports.Pool, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", "schedule not found")
			return
		}
		reports.writeServiceError(w, err)
		return
	}
	if reports.AuthorizeCustomerAccess != nil {
		if err := reports.AuthorizeCustomerAccess(r, row.CustomerID); err != nil {
			reports.writeServiceError(w, err)
			return
		}
	}
	httpresponse.JSON(w, http.StatusOK, row)
}

func (reports *ReportsHTTPHandlers) updateReportSchedule(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	req, err := coldpath.DecodeRequest[UpdateReportScheduleRequest](w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}
	if req.ReportKey == "" || req.CronExpr == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "report_key and cron_expr required")
		return
	}
	existing, err := getReportSchedule(r.Context(), reports.Pool, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", "schedule not found")
			return
		}
		reports.writeServiceError(w, err)
		return
	}
	if reports.AuthorizeCustomerAccess != nil {
		if err := reports.AuthorizeCustomerAccess(r, existing.CustomerID); err != nil {
			reports.writeServiceError(w, err)
			return
		}
	}
	updated, err := updateReportSchedule(r.Context(), reports.Pool, id, req)
	if err != nil {
		if errors.Is(err, errInvalidCronExpr) || err.Error() == "invalid cron_expr" {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			return
		}
		reports.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, updated)
}

func (reports *ReportsHTTPHandlers) deleteReportSchedule(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	existing, err := getReportSchedule(r.Context(), reports.Pool, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", "schedule not found")
			return
		}
		reports.writeServiceError(w, err)
		return
	}
	if reports.AuthorizeCustomerAccess != nil {
		if err := reports.AuthorizeCustomerAccess(r, existing.CustomerID); err != nil {
			reports.writeServiceError(w, err)
			return
		}
	}
	if err := deleteReportSchedule(r.Context(), reports.Pool, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", "schedule not found")
			return
		}
		reports.writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
