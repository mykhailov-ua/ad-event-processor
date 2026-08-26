package controlplane

import (
	"net/http"

	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
)

type AutomationHTTPHandlers struct {
	Service           *Service
	ApplyRateLimit    func(http.HandlerFunc) http.HandlerFunc
	RequirePermission func(string, http.HandlerFunc) http.HandlerFunc
}

func (h *AutomationHTTPHandlers) Register(mux *http.ServeMux) {
	if h == nil || h.Service == nil {
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
	mux.HandleFunc("GET /api/v1/automation/rules", limit(perm("campaigns:read", h.listRules)))
	mux.HandleFunc("POST /api/v1/automation/rules", limit(perm("campaigns:write", h.createRule)))
	mux.HandleFunc("PUT /api/v1/automation/rules/{id}", limit(perm("campaigns:write", h.updateRule)))
	mux.HandleFunc("DELETE /api/v1/automation/rules/{id}", limit(perm("campaigns:write", h.deleteRule)))
	mux.HandleFunc("POST /api/v1/automation/rules/{id}/dry-run", limit(perm("campaigns:read", h.dryRun)))
}

func (h *AutomationHTTPHandlers) listRules(w http.ResponseWriter, r *http.Request) {
	custStr := r.URL.Query().Get("customer_id")
	custID, err := uuid.Parse(custStr)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid customer_id")
		return
	}
	rules, err := h.Service.ListAutomationRules(r.Context(), custID)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if rules == nil {
		rules = []AutomationRuleDTO{}
	}
	httpresponse.JSON(w, http.StatusOK, rules)
}

func (h *AutomationHTTPHandlers) createRule(w http.ResponseWriter, r *http.Request) {
	req, ok := coldpath.DecodeRequestOrBadRequest[UpsertAutomationRuleRequest](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	rule, err := h.Service.CreateAutomationRule(r.Context(), req)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusCreated, rule)
}

func (h *AutomationHTTPHandlers) updateRule(w http.ResponseWriter, r *http.Request) {
	ruleID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid rule id")
		return
	}
	req, ok := coldpath.DecodeRequestOrBadRequest[UpsertAutomationRuleRequest](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	rule, err := h.Service.UpdateAutomationRule(r.Context(), ruleID, req)
	if err != nil {
		if err.Error() == "rule not found" {
			httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error())
			return
		}
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, rule)
}

func (h *AutomationHTTPHandlers) deleteRule(w http.ResponseWriter, r *http.Request) {
	ruleID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid rule id")
		return
	}
	if err := h.Service.DeleteAutomationRule(r.Context(), ruleID); err != nil {
		if err.Error() == "rule not found" {
			httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error())
			return
		}
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AutomationHTTPHandlers) dryRun(w http.ResponseWriter, r *http.Request) {
	ruleID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid rule id")
		return
	}
	resp, err := h.Service.DryRunAutomationRule(r.Context(), ruleID)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, resp)
}
