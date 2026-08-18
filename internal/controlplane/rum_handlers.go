package controlplane

import (
	"encoding/json"
	"net/http"

	"github.com/bidshard/ad-event-processor/pkg/coldpath"
	"github.com/bidshard/ad-event-processor/pkg/httpresponse"
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

func (ops *OpsHTTPHandlers) registerRUMRoutes(mux *http.ServeMux) {
	if ops == nil || ops.RUMStore == nil {
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
	mux.HandleFunc("POST /api/v1/ops/rum", limit(ops.postClientRUM))
	mux.HandleFunc("GET /api/v1/ops/rum", limit(perm("shards:read", ops.getClientRUM)))
}

func (ops *OpsHTTPHandlers) postClientRUM(w http.ResponseWriter, r *http.Request) {
	if ops.RUMStore == nil {
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
	ops.RUMStore.AppendClientRUM(req)
	w.WriteHeader(http.StatusAccepted)
}

func (ops *OpsHTTPHandlers) getClientRUM(w http.ResponseWriter, r *http.Request) {
	if ops.RUMStore == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "rum store unavailable")
		return
	}
	httpresponse.JSON(w, http.StatusOK, map[string]any{
		"events": ops.RUMStore.SnapshotClientRUM(),
	})
}
