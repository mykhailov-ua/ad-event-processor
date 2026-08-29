package opsadmin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"ad-event-processor/internal/fraudadmin"
	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"
	"ad-event-processor/pkg/supportbundle"
)

func (h *HTTPHandlers) registerDashboardRoutes(mux *http.ServeMux) {
	if h == nil || h.OpsReader == nil {
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
	mux.HandleFunc("GET /api/v1/ops/dashboard/summary", limit(perm("shards:read", h.getDashboardSummary)))
	if h.OpsReader != nil {
		mux.HandleFunc("GET /api/v1/ops/domains/rotation", limit(perm("settings:read", h.listDomainRotation)))
	}
	mux.HandleFunc("GET /api/v1/ops/dashboard/metrics", limit(perm("shards:read", h.getDashboardMetrics)))
	mux.HandleFunc("GET /api/v1/ops/dashboard/stream", limit(perm("shards:read", h.streamDashboard)))
}

func (h *HTTPHandlers) getDashboardSummary(w http.ResponseWriter, r *http.Request) {
	summary, err := h.OpsReader.GetDashboardSummary(r.Context())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, summary)
}

func (h *HTTPHandlers) listDomainRotation(w http.ResponseWriter, r *http.Request) {
	result, err := h.OpsReader.ListDomainRotation(r.Context())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, result)
}

func (h *HTTPHandlers) getDashboardMetrics(w http.ResponseWriter, r *http.Request) {
	rangeHours := 24
	if raw := r.URL.Query().Get("range"); raw != "" {
		if len(raw) >= 2 && raw[len(raw)-1] == 'h' {
			if n, err := strconv.Atoi(raw[:len(raw)-1]); err == nil && n > 0 {
				rangeHours = n
			}
		}
	}
	metricName := r.URL.Query().Get("name")
	metrics, err := h.OpsReader.GetDashboardMetrics(r.Context(), rangeHours, metricName)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, metrics)
}

func (h *HTTPHandlers) streamDashboard(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		httpresponse.Error(w, http.StatusInternalServerError, "STREAM_UNSUPPORTED", "streaming not supported")
		return
	}
	ctx := r.Context()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	writeEvent := func() bool {
		summary, err := h.OpsReader.GetDashboardSummary(ctx)
		if err != nil {
			return true
		}
		payload, err := json.Marshal(map[string]any{
			"generated_at": summary.GeneratedAt,
			"data":         summary,
		})
		if err != nil {
			return true
		}
		if _, err := fmt.Fprintf(w, "event: dashboard\ndata: %s\n\n", payload); err != nil {
			return false
		}
		flusher.Flush()
		return true
	}

	if !writeEvent() {
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !writeEvent() {
				return
			}
		}
	}
}

func (h *HTTPHandlers) registerMLModelRoutes(mux *http.ServeMux) {
	if h == nil || h.OpsReader == nil {
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
	mux.HandleFunc("GET /api/v1/ops/ml-model", limit(perm("shards:read", h.getMLModelStatus)))
	mux.HandleFunc("GET /api/v1/ops/ml-model/eval", limit(perm("shards:read", h.getMLEvalReport)))
	mux.HandleFunc("GET /api/v1/ops/ml-model/labels", limit(perm("shards:read", h.listMLManualLabels)))
	mux.HandleFunc("POST /api/v1/ops/ml-model/labels", limit(perm("shards:write", h.postMLManualLabel)))
}

func (h *HTTPHandlers) getMLModelStatus(w http.ResponseWriter, r *http.Request) {
	status, err := h.OpsReader.GetMLModelStatus(r.Context())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, status)
}

func (h *HTTPHandlers) getMLEvalReport(w http.ResponseWriter, r *http.Request) {
	report, err := h.OpsReader.GetMLEvalReport(r.Context())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, report)
}

func (h *HTTPHandlers) listMLManualLabels(w http.ResponseWriter, r *http.Request) {
	labels, err := h.OpsReader.ListMLManualLabels(r.Context())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	if labels == nil {
		labels = []MLManualLabelDTO{}
	}
	httpresponse.JSON(w, http.StatusOK, labels)
}

func (h *HTTPHandlers) postMLManualLabel(w http.ResponseWriter, r *http.Request) {
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}
	req, err := coldpath.DecodeBody[MLManualLabelRequest](body)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if req.IPHash == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "ip_hash required")
		return
	}
	if !fraudadmin.ValidMLIPHashHex(req.IPHash) {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "ip_hash must be 32 hex characters")
		return
	}
	if req.Label != 0 && req.Label != 1 {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "label must be 0 or 1")
		return
	}
	if err := h.OpsReader.AddMLManualLabel(r.Context(), req.IPHash, req.Label, req.Reason); err != nil {
		h.writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (h *HTTPHandlers) RegisterSupportBundleRoutes(mux *http.ServeMux) {
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

func (h *HTTPHandlers) registerSupportBundleRoutes(mux *http.ServeMux) {
	h.RegisterSupportBundleRoutes(mux)
}

func (h *HTTPHandlers) postSupportBundle(w http.ResponseWriter, r *http.Request) {
	if h.SupportBundle == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "BUNDLE_UNAVAILABLE", "support bundle not configured")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), supportbundle.DefaultTimeout)
	defer cancel()

	w.Header().Set("Content-Type", "application/gzip")
	w.Header().Set("Content-Disposition", `attachment; filename="ad-event-processor-support-bundle.tar.gz"`)
	w.Header().Set("Cache-Control", "no-store")

	if err := h.SupportBundle.WriteSupportBundle(ctx, w); err != nil {
		h.writeServiceError(w, err)
		return
	}
}
