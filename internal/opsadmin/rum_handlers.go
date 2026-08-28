package opsadmin

import (
	"encoding/json"
	"net/http"

	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"
)

type ClientRUMIngestDTO struct {
	Path   string          `json:"path,omitempty"`
	Vitals json.RawMessage `json:"vitals,omitempty"`
	API    json.RawMessage `json:"api,omitempty"`
	Guards json.RawMessage `json:"guards,omitempty"`
	Probes json.RawMessage `json:"probes,omitempty"`
	Memory json.RawMessage `json:"memory,omitempty"`
}

type RUMStore interface {
	AppendClientRUM(ev ClientRUMIngestDTO)
	SnapshotClientRUM() []any
}

func (h *HTTPHandlers) registerRUMRoutes(mux *http.ServeMux) {
	if h == nil || h.RUMStore == nil {
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
	mux.HandleFunc("POST /api/v1/ops/rum", limit(h.postClientRUM))
	mux.HandleFunc("GET /api/v1/ops/rum", limit(perm("shards:read", h.getClientRUM)))
}

func (h *HTTPHandlers) PostClientRUM(w http.ResponseWriter, r *http.Request) {
	h.postClientRUM(w, r)
}

func (h *HTTPHandlers) GetClientRUM(w http.ResponseWriter, r *http.Request) {
	h.getClientRUM(w, r)
}

func (h *HTTPHandlers) postClientRUM(w http.ResponseWriter, r *http.Request) {
	if h.RUMStore == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "rum store unavailable")
		return
	}
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}
	var req ClientRUMIngestDTO
	if err := json.Unmarshal(body, &req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json body")
		return
	}
	h.RUMStore.AppendClientRUM(req)
	w.WriteHeader(http.StatusAccepted)
}

func (h *HTTPHandlers) getClientRUM(w http.ResponseWriter, r *http.Request) {
	if h.RUMStore == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "rum store unavailable")
		return
	}
	httpresponse.JSON(w, http.StatusOK, map[string]any{
		"events": h.RUMStore.SnapshotClientRUM(),
	})
}
