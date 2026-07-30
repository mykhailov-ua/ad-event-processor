package adminapi

import (
	"log/slog"
	"net/http"

	"espx/pkg/httpresponse"
)

type RolesReloader interface {
	ReloadRoles() error
	RolesPath() string
}

func (h *OpsHTTPHandlers) registerRolesRoutes(mux *http.ServeMux) {
	if h.RolesReloader == nil {
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
	mux.HandleFunc("POST /api/v1/ops/roles/reload", limit(perm("settings:write", h.reloadRoles)))
}

func (h *OpsHTTPHandlers) reloadRoles(w http.ResponseWriter, r *http.Request) {
	if h.RolesReloader == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "roles reloader not configured")
		return
	}
	if err := h.RolesReloader.ReloadRoles(); err != nil {
		slog.Error("roles reload failed", "error", err)
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL", "failed to reload roles")
		return
	}
	httpresponse.JSON(w, http.StatusOK, map[string]string{"status": "reloaded", "path": h.RolesReloader.RolesPath()})
}
