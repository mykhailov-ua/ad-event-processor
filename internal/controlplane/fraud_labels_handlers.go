package controlplane

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
)

type FraudLabelsService interface {
	ListMLManualLabelsForCustomer(ctx context.Context, customerID uuid.UUID, limit int) ([]MLManualLabelDTO, error)
	UpsertMLManualLabelForCustomer(ctx context.Context, customerID uuid.UUID, ipHash string, label int, reason string) error
	BulkUpsertMLManualLabelsForCustomer(ctx context.Context, customerID uuid.UUID, rows []FraudManualLabelRow) (int, error)
}

type FraudManualLabelRow struct {
	IPHash string `json:"ip_hash"`
	Label  int    `json:"label"`
	Reason string `json:"reason"`
}

type FraudManualLabelRequest struct {
	IPHash string `json:"ip_hash"`
	Label  int    `json:"label"`
	Reason string `json:"reason"`
}

type FraudManualLabelBulkRequest struct {
	Rows []FraudManualLabelRow `json:"rows"`
}

type FraudManualLabelBulkResponse struct {
	Upserted int `json:"upserted"`
}

type FraudHTTPHandlers struct {
	Labels                  FraudLabelsService
	Decisions               FraudDecisionsService
	Integrations            FraudIntegrationsService
	Overrides               FraudOverridesService
	Presets                 FraudPresetsService
	ApplyRateLimit          func(http.HandlerFunc) http.HandlerFunc
	AllowFraudDecision      func(customerID string) bool
	RequirePermission       func(string, http.HandlerFunc) http.HandlerFunc
	RequireAnyPermission    func([]string, http.HandlerFunc) http.HandlerFunc
	ResolveCustomerID       func(*http.Request, *uuid.UUID) (uuid.UUID, error)
	AuthorizeCampaignAccess func(*http.Request, uuid.UUID) error
	WriteServiceError       func(http.ResponseWriter, error)
}

func (h *FraudHTTPHandlers) Register(mux *http.ServeMux) {
	if h == nil {
		return
	}
	limit := h.ApplyRateLimit
	perm := h.RequirePermission
	permAny := h.RequireAnyPermission
	if limit == nil {
		limit = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	if perm == nil {
		perm = func(_ string, next http.HandlerFunc) http.HandlerFunc { return next }
	}
	if permAny == nil {
		permAny = func(_ []string, next http.HandlerFunc) http.HandlerFunc { return next }
	}

	if h.Labels != nil {
		mux.HandleFunc("GET /api/v1/fraud/labels", limit(perm("audit:read", h.listFraudLabels)))
		mux.HandleFunc("POST /api/v1/fraud/labels", limit(permAny([]string{"campaigns:write", "shards:write"}, h.postFraudLabel)))
		mux.HandleFunc("POST /api/v1/fraud/labels/bulk", limit(permAny([]string{"campaigns:write", "shards:write"}, h.postFraudLabelsBulk)))
	}
	h.registerFraudDecisionRoutes(mux, limit, perm)
	h.registerFraudIntegrationRoutes(mux, limit, perm)
	h.registerFraudOverrideRoutes(mux, limit, permAny)
	h.registerFraudPresetRoutes(mux, limit, permAny)
}

func (h *FraudHTTPHandlers) listFraudLabels(w http.ResponseWriter, r *http.Request) {
	customerID, ok := h.resolveCustomerID(w, r)
	if !ok {
		return
	}
	limit := fraudManualLabelsDefaultLimit
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid limit")
			return
		}
		limit = parsed
	}
	if limit > fraudManualLabelsMaxLimit {
		limit = fraudManualLabelsMaxLimit
	}

	labels, err := h.Labels.ListMLManualLabelsForCustomer(r.Context(), customerID, limit)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	if labels == nil {
		labels = []MLManualLabelDTO{}
	}
	httpresponse.JSON(w, http.StatusOK, labels)
}

func (h *FraudHTTPHandlers) postFraudLabel(w http.ResponseWriter, r *http.Request) {
	customerID, ok := h.resolveCustomerID(w, r)
	if !ok {
		return
	}
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}
	req, err := coldpath.DecodeBody[FraudManualLabelRequest](body)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
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
	if err := h.Labels.UpsertMLManualLabelForCustomer(r.Context(), customerID, strings.ToLower(req.IPHash), req.Label, req.Reason); err != nil {
		h.writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (h *FraudHTTPHandlers) postFraudLabelsBulk(w http.ResponseWriter, r *http.Request) {
	customerID, ok := h.resolveCustomerID(w, r)
	if !ok {
		return
	}
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}
	req, err := coldpath.DecodeBody[FraudManualLabelBulkRequest](body)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if len(req.Rows) == 0 {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "rows required")
		return
	}
	if len(req.Rows) > fraudManualLabelsBulkMax {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "max 500 rows per bulk request")
		return
	}
	rows := make([]FraudManualLabelRow, len(req.Rows))
	for i, row := range req.Rows {
		if !validMLIPHashHex(row.IPHash) {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "ip_hash must be 32 hex characters")
			return
		}
		if row.Label != 0 && row.Label != 1 {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "label must be 0 or 1")
			return
		}
		rows[i] = FraudManualLabelRow{
			IPHash: strings.ToLower(row.IPHash),
			Label:  row.Label,
			Reason: row.Reason,
		}
	}
	upserted, err := h.Labels.BulkUpsertMLManualLabelsForCustomer(r.Context(), customerID, rows)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, FraudManualLabelBulkResponse{Upserted: upserted})
}

func (h *FraudHTTPHandlers) resolveCustomerID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	var customerID uuid.UUID
	if custStr := r.URL.Query().Get("customer_id"); custStr != "" {
		id, err := uuid.Parse(custStr)
		if err != nil {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid customer_id")
			return uuid.Nil, false
		}
		customerID = id
	}
	if h.ResolveCustomerID != nil {
		resolved, err := h.ResolveCustomerID(r, nonNilUUID(customerID))
		if err != nil {
			h.writeServiceError(w, err)
			return uuid.Nil, false
		}
		customerID = resolved
	}
	if customerID == uuid.Nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "customer_id is required")
		return uuid.Nil, false
	}
	return customerID, true
}

func (h *FraudHTTPHandlers) writeServiceError(w http.ResponseWriter, err error) {
	if h.WriteServiceError != nil {
		h.WriteServiceError(w, err)
		return
	}
	httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL", "internal error")
}

const (
	fraudManualLabelsDefaultLimit = 50
	fraudManualLabelsMaxLimit     = 100
	fraudManualLabelsBulkMax      = 500
	fraudThreatBatchMax           = 500
)
