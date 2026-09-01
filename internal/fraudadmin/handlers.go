package fraudadmin

import (
	"net/http"
	"strconv"
	"strings"

	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
)

type HTTPHandlers struct {
	Labels                  LabelsService
	Decisions               DecisionsService
	Integrations            IntegrationsService
	Overrides               OverridesService
	Presets                 PresetsService
	ApplyRateLimit          func(http.HandlerFunc) http.HandlerFunc
	AllowFraudDecision      func(customerID string) bool
	RequirePermission       func(string, http.HandlerFunc) http.HandlerFunc
	RequireAnyPermission    func([]string, http.HandlerFunc) http.HandlerFunc
	ResolveCustomerID       func(*http.Request, *uuid.UUID) (uuid.UUID, error)
	AuthorizeCampaignAccess func(*http.Request, uuid.UUID) error
	WriteServiceError       func(http.ResponseWriter, error)
}

func (h *HTTPHandlers) Register(mux *http.ServeMux) {
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

func (h *HTTPHandlers) listFraudLabels(w http.ResponseWriter, r *http.Request) {
	customerID, ok := h.resolveCustomerID(w, r)
	if !ok {
		return
	}
	limit := ManualLabelsDefaultLimit
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid limit")
			return
		}
		limit = parsed
	}
	if limit > ManualLabelsMaxLimit {
		limit = ManualLabelsMaxLimit
	}

	offset := 0
	if raw := strings.TrimSpace(r.URL.Query().Get("offset")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid offset")
			return
		}
		offset = parsed
	}

	labels, total, err := h.Labels.ListMLManualLabelsForCustomer(r.Context(), customerID, limit, offset)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	if labels == nil {
		labels = []MLManualLabelDTO{}
	}
	httpresponse.JSON(w, http.StatusOK, FraudLabelsListResponse{
		Items:  labels,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	})
}

func (h *HTTPHandlers) postFraudLabel(w http.ResponseWriter, r *http.Request) {
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
	if !ValidMLIPHashHex(req.IPHash) {
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

func (h *HTTPHandlers) postFraudLabelsBulk(w http.ResponseWriter, r *http.Request) {
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
	if len(req.Rows) > ManualLabelsBulkMax {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "max 500 rows per bulk request")
		return
	}
	rows := make([]FraudManualLabelRow, len(req.Rows))
	for i, row := range req.Rows {
		if !ValidMLIPHashHex(row.IPHash) {
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

func (h *HTTPHandlers) resolveCustomerID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
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

func (h *HTTPHandlers) writeServiceError(w http.ResponseWriter, err error) {
	if h.WriteServiceError != nil {
		h.WriteServiceError(w, err)
		return
	}
	httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL", "internal error")
}

func nonNilUUID(id uuid.UUID) *uuid.UUID {
	if id == uuid.Nil {
		return nil
	}
	return &id
}
