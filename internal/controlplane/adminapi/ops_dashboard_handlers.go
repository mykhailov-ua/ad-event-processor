package adminapi

import (
	"net/http"
	"strconv"

	"espx/pkg/httpresponse"
)

func (ops *OpsHTTPHandlers) registerDashboardRoutes(mux *http.ServeMux) {
	if ops == nil || ops.OpsReader == nil {
		return
	}
	limit := ops.ApplyRateLimit
	perm := ops.RequirePermission
	if limit == nil {
		limit = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	if perm == nil {
		perm = func(_ string, next http.HandlerFunc) http.HandlerFunc { return next }
	}
	mux.HandleFunc("GET /api/v1/ops/dashboard/summary", limit(perm("shards:read", ops.getDashboardSummary)))
	mux.HandleFunc("GET /api/v1/ops/dashboard/metrics", limit(perm("shards:read", ops.getDashboardMetrics)))
}

func (ops *OpsHTTPHandlers) getDashboardSummary(w http.ResponseWriter, r *http.Request) {
	summary, err := ops.OpsReader.GetDashboardSummary(r.Context())
	if err != nil {
		ops.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, summary)
}

func (ops *OpsHTTPHandlers) getDashboardMetrics(w http.ResponseWriter, r *http.Request) {
	rangeHours := 24
	if raw := r.URL.Query().Get("range"); raw != "" {
		if len(raw) >= 2 && raw[len(raw)-1] == 'h' {
			if n, err := strconv.Atoi(raw[:len(raw)-1]); err == nil && n > 0 {
				rangeHours = n
			}
		}
	}
	metricName := r.URL.Query().Get("name")
	metrics, err := ops.OpsReader.GetDashboardMetrics(r.Context(), rangeHours, metricName)
	if err != nil {
		ops.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, metrics)
}
