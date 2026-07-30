package adminapi

import (
	"context"
	"log/slog"
	"net/http"

	"espx/internal/billing/plansyaml"

	"espx/pkg/httpresponse"
)

type PlansReloader interface {
	ReloadPlans(ctx context.Context, dryRun bool) (plansyaml.ReloadReport, error)
	PlansPath() string
}

func (h *OpsHTTPHandlers) registerPlansRoutes(mux *http.ServeMux) {
	if h.PlansReloader == nil {
		return
	}
	limit := h.ApplyRateLimit
	perm := h.RequirePermission
	if limit == nil {
		limit = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	if perm == nil {
		perm = func(_ string, next http.HandlerFunc) http.HandlerFunc { return next }
	}
	mux.HandleFunc("POST /api/v1/ops/plans/reload", limit(perm("ops:write", h.reloadPlans)))
}

func (h *OpsHTTPHandlers) reloadPlans(w http.ResponseWriter, r *http.Request) {
	if h.PlansReloader == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "plans reloader not configured")
		return
	}
	dryRun := r.URL.Query().Get("dry_run") == "1" || r.Header.Get("X-Dry-Run") == "1"
	report, err := h.PlansReloader.ReloadPlans(r.Context(), dryRun)
	if err != nil {
		slog.Error("plans reload failed", "error", err)
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, report)
}
