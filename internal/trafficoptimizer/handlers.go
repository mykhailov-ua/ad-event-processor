package trafficoptimizer

import (
	"context"
	"errors"
	"net/http"

	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
)

type RulesAdmin interface {
	ListRules(ctx context.Context, customerID uuid.UUID) ([]RuleDTO, error)
	CreateRule(ctx context.Context, req UpsertRuleRequest) (RuleDTO, error)
	UpdateRule(ctx context.Context, ruleID uuid.UUID, req UpsertRuleRequest) (RuleDTO, error)
	DeleteRule(ctx context.Context, ruleID uuid.UUID) error
	DryRunRule(ctx context.Context, ruleID uuid.UUID) (DryRunResponse, error)
}

type HTTPHandlers struct {
	Rules             RulesAdmin
	ApplyRateLimit    func(http.HandlerFunc) http.HandlerFunc
	RequirePermission func(string, http.HandlerFunc) http.HandlerFunc
}

func (h *HTTPHandlers) Register(mux *http.ServeMux) {
	if h == nil || h.Rules == nil {
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
	mux.HandleFunc("GET /api/v1/traffic-optimizer/presets", limit(perm("campaigns:read", h.listPresets)))
	mux.HandleFunc("GET /api/v1/traffic-optimizer/rules", limit(perm("campaigns:read", h.listRules)))
	mux.HandleFunc("POST /api/v1/traffic-optimizer/rules", limit(perm("campaigns:write", h.createRule)))
	mux.HandleFunc("PUT /api/v1/traffic-optimizer/rules/{id}", limit(perm("campaigns:write", h.updateRule)))
	mux.HandleFunc("DELETE /api/v1/traffic-optimizer/rules/{id}", limit(perm("campaigns:write", h.deleteRule)))
	mux.HandleFunc("POST /api/v1/traffic-optimizer/rules/{id}/dry-run", limit(perm("campaigns:read", h.dryRun)))
}

func (h *HTTPHandlers) listPresets(w http.ResponseWriter, _ *http.Request) {
	httpresponse.JSON(w, http.StatusOK, ListPresets())
}

func (h *HTTPHandlers) listRules(w http.ResponseWriter, r *http.Request) {
	custStr := r.URL.Query().Get("customer_id")
	custID, err := uuid.Parse(custStr)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid customer_id")
		return
	}
	rules, err := h.Rules.ListRules(r.Context(), custID)
	if err != nil {
		if errors.Is(err, ErrUnavailable) {
			httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", err.Error())
			return
		}
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if rules == nil {
		rules = []RuleDTO{}
	}
	httpresponse.JSON(w, http.StatusOK, rules)
}

func (h *HTTPHandlers) createRule(w http.ResponseWriter, r *http.Request) {
	req, ok := coldpath.DecodeRequestOrBadRequest[UpsertRuleRequest](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	rule, err := h.Rules.CreateRule(r.Context(), req)
	if err != nil {
		writeRuleServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusCreated, rule)
}

func (h *HTTPHandlers) updateRule(w http.ResponseWriter, r *http.Request) {
	ruleID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid rule id")
		return
	}
	req, ok := coldpath.DecodeRequestOrBadRequest[UpsertRuleRequest](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	rule, err := h.Rules.UpdateRule(r.Context(), ruleID, req)
	if err != nil {
		writeRuleServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, rule)
}

func (h *HTTPHandlers) deleteRule(w http.ResponseWriter, r *http.Request) {
	ruleID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid rule id")
		return
	}
	if err := h.Rules.DeleteRule(r.Context(), ruleID); err != nil {
		writeRuleServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *HTTPHandlers) dryRun(w http.ResponseWriter, r *http.Request) {
	ruleID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid rule id")
		return
	}
	resp, err := h.Rules.DryRunRule(r.Context(), ruleID)
	if err != nil {
		writeRuleServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, resp)
}

func writeRuleServiceError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrNotImplemented) {
		httpresponse.Error(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", NotImplementedMessage)
		return
	}
	if errors.Is(err, ErrUnavailable) {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", err.Error())
		return
	}
	httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
}
