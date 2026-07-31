package adminapi

import (
	"encoding/json"
	"net/http"
	"unicode"

	"espx/pkg/coldpath"
	"espx/pkg/httpresponse"
)

type MLModelVersionDTO struct {
	ID               string          `json:"id"`
	ArtifactHash     string          `json:"artifact_hash"`
	Status           string          `json:"status"`
	CreatedAt        string          `json:"created_at"`
	ArtifactMetadata json.RawMessage `json:"artifact_metadata,omitempty"`
}

type MLModelRedisDTO struct {
	VersionID        string `json:"version_id,omitempty"`
	Hash             string `json:"hash,omitempty"`
	AppliedAt        string `json:"applied_at,omitempty"`
	ShardsReporting  int    `json:"shards_reporting"`
	ShardsConsistent bool   `json:"shards_consistent"`
}

type MLShardSyncDTO struct {
	ShardID      int    `json:"shard_id"`
	ModelVersion string `json:"model_version"`
	Phase        string `json:"phase"`
	StartedAt    string `json:"started_at"`
}

type MLFeatureImportanceDTO struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
}

type MLModelStatusDTO struct {
	ActiveVersion  *MLModelVersionDTO     `json:"active_version,omitempty"`
	SyncingVersion *MLModelVersionDTO     `json:"syncing_version,omitempty"`
	Redis          MLModelRedisDTO        `json:"redis"`
	ShardSync      []MLShardSyncDTO       `json:"shard_sync"`
	Drift          json.RawMessage        `json:"drift,omitempty"`
	DriftDetected  bool                   `json:"drift_detected"`
	Precision      float64                `json:"precision,omitempty"`
	Recall         float64                `json:"recall,omitempty"`
	Importance     []MLFeatureImportanceDTO `json:"importance,omitempty"`
}

type MLManualLabelRequest struct {
	IPHash string `json:"ip_hash"`
	Label  int    `json:"label"`
	Reason string `json:"reason"`
}

type MLManualLabelDTO struct {
	IPHash    string `json:"ip_hash"`
	Label     int    `json:"label"`
	Reason    string `json:"reason"`
	Source    string `json:"source"`
	CreatedAt string `json:"created_at"`
}

func (ops *OpsHTTPHandlers) registerMLModelRoutes(mux *http.ServeMux) {
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
	mux.HandleFunc("GET /api/v1/ops/ml-model", limit(perm("shards:read", ops.getMLModelStatus)))
	mux.HandleFunc("GET /api/v1/ops/ml-model/labels", limit(perm("shards:read", ops.listMLManualLabels)))
	mux.HandleFunc("POST /api/v1/ops/ml-model/labels", limit(perm("shards:write", ops.postMLManualLabel)))
}

func (ops *OpsHTTPHandlers) getMLModelStatus(w http.ResponseWriter, r *http.Request) {
	status, err := ops.OpsReader.GetMLModelStatus(r.Context())
	if err != nil {
		ops.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, status)
}

func (ops *OpsHTTPHandlers) listMLManualLabels(w http.ResponseWriter, r *http.Request) {
	labels, err := ops.OpsReader.ListMLManualLabels(r.Context())
	if err != nil {
		ops.writeServiceError(w, err)
		return
	}
	if labels == nil {
		labels = []MLManualLabelDTO{}
	}
	httpresponse.JSON(w, http.StatusOK, labels)
}

func (ops *OpsHTTPHandlers) postMLManualLabel(w http.ResponseWriter, r *http.Request) {
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
	if !validMLIPHashHex(req.IPHash) {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "ip_hash must be 32 hex characters")
		return
	}
	if req.Label != 0 && req.Label != 1 {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "label must be 0 or 1")
		return
	}
	if err := ops.OpsReader.AddMLManualLabel(r.Context(), req.IPHash, req.Label, req.Reason); err != nil {
		ops.writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func validMLIPHashHex(s string) bool {
	if len(s) != 32 {
		return false
	}
	for _, c := range s {
		if unicode.IsDigit(c) {
			continue
		}
		if (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') {
			continue
		}
		return false
	}
	return true
}
