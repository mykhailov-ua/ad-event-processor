package management

import (
	"encoding/json"
	"io"
	"net/http"

	"espx/pkg/httpresponse"

	"github.com/google/uuid"
)

type regionIngestBatchJSON struct {
	RegionCode  uint8  `json:"region_code"`
	NodeID      string `json:"node_id"`
	SourceEpoch uint32 `json:"source_epoch"`
	Seq         uint64 `json:"seq"`
	FactorU     string `json:"factor_u"`
	Payload     []byte `json:"payload"`
	OpID        string `json:"op_id,omitempty"`
}

// registerRegionIngestRoutes mounts global D3 ingest for region-proxy uplink.
func (h *Handler) registerRegionIngestRoutes(mux *http.ServeMux) {
	if h.cfg == nil || !h.cfg.MultiRegionGlobal() {
		return
	}
	mux.HandleFunc("POST /api/v1/region/ingest/batch", h.pgHigh(h.postRegionIngestBatch))
}

// postRegionIngestBatch handles POST /api/v1/region/ingest/batch from region-proxy uplink.
func (h *Handler) postRegionIngestBatch(w http.ResponseWriter, r *http.Request) {
	key := r.Header.Get("X-Admin-API-Key")
	if key == "" || h.cfg == nil || key != string(h.cfg.AdminAPIKey) {
		httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 4*1024*1024))
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid body")
		return
	}
	var in regionIngestBatchJSON
	if err := json.Unmarshal(body, &in); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json")
		return
	}
	factorU, err := uuid.Parse(in.FactorU)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid factor_u")
		return
	}
	var opID uuid.UUID
	if in.OpID != "" {
		opID, err = uuid.Parse(in.OpID)
		if err != nil {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid op_id")
			return
		}
	}
	result, err := h.svc.IngestRegionProxyBatch(r.Context(), RegionIngestBatchInput{
		RegionCode:  in.RegionCode,
		NodeID:      in.NodeID,
		SourceEpoch: in.SourceEpoch,
		Seq:         in.Seq,
		FactorU:     factorU,
		Payload:     in.Payload,
		OpID:        opID,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, map[string]string{
		"outcome":   string(result.Outcome),
		"dedup_key": result.DedupKey,
	})
}
