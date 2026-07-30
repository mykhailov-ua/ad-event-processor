package adminapi

import (
	"context"
	"io"
	"net/http"

	"espx/pkg/httpresponse"
	"espx/pkg/supportbundle"
)

type SupportBundleWriter interface {
	WriteSupportBundle(ctx context.Context, w io.Writer) error
}

func (h *OpsHTTPHandlers) registerSupportBundleRoutes(mux *http.ServeMux) {
	if h == nil || h.SupportBundle == nil {
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
	mux.HandleFunc("POST /api/v1/ops/support/bundle", limit(perm("ops:write", h.postSupportBundle)))
}

func (h *OpsHTTPHandlers) postSupportBundle(w http.ResponseWriter, r *http.Request) {
	if h.SupportBundle == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "BUNDLE_UNAVAILABLE", "support bundle not configured")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), supportbundle.DefaultTimeout)
	defer cancel()

	w.Header().Set("Content-Type", "application/gzip")
	w.Header().Set("Content-Disposition", `attachment; filename="espx-support-bundle.tar.gz"`)
	w.Header().Set("Cache-Control", "no-store")

	if err := h.SupportBundle.WriteSupportBundle(ctx, w); err != nil {
		h.writeServiceError(w, err)
		return
	}
}
