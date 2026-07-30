package management

import (
	"log/slog"
	"net/http"

	"espx/internal/management/authz"
	"espx/pkg/httpresponse"
)

func (h *Handler) registerOpsRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /admin/ops/shards", h.limit(h.perm(h.getShardHealth, PermShardsRead)))
	mux.HandleFunc("POST /api/v1/ops/roles/reload", h.limit(h.perm(h.reloadRoles, PermSettingsWrite)))
}

func (h *Handler) getShardHealth(w http.ResponseWriter, r *http.Request) {
	report, err := h.svc.GetShardHealth(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, report)
}

func (h *Handler) reloadRoles(w http.ResponseWriter, r *http.Request) {
	if h.authMiddleware == nil || h.authMiddleware.policy == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "policy store not configured")
		return
	}
	path := authz.DefaultRolesPath()
	if err := authz.LoadRolesYAML(path, h.authMiddleware.policy); err != nil {
		slog.Error("roles reload failed", "path", path, "error", err)
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL", "failed to reload roles")
		return
	}
	httpresponse.JSON(w, http.StatusOK, map[string]string{"status": "reloaded", "path": path})
}
