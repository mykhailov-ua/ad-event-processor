package adminapi

import (
	"context"
	"net/http"
	"strconv"

	"espx/pkg/httpresponse"
)

type AuditLister interface {
	ListAuditLogs(ctx context.Context, limit, offset int32, redactPII bool) (any, int64, error)
}

func (h *OpsHTTPHandlers) registerAuditRoutes(mux *http.ServeMux) {
	if h.AuditLister == nil {
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
	mux.HandleFunc("GET /api/v1/audit", limit(perm("audit:read", h.listAudit)))
}

func (h *OpsHTTPHandlers) listAudit(w http.ResponseWriter, r *http.Request) {
	limit, offset := parseAPIPagination(r)
	redact := r.URL.Query().Get("redact_pii") == "true"
	logs, total, err := h.AuditLister.ListAuditLogs(r.Context(), limit, offset, redact)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	w.Header().Set("X-Total-Count", strconv.FormatInt(total, 10))
	httpresponse.JSON(w, http.StatusOK, logs)
}
