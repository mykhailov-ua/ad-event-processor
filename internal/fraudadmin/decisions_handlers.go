package fraudadmin

import (
	"net/http"
	"strconv"
	"strings"

	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
)

func (h *HTTPHandlers) registerFraudDecisionRoutes(mux *http.ServeMux, limit func(http.HandlerFunc) http.HandlerFunc, perm func(string, http.HandlerFunc) http.HandlerFunc) {
	if h == nil || h.Decisions == nil {
		return
	}
	mux.HandleFunc("GET /api/v1/fraud/decisions", limit(perm("audit:read", h.getFraudDecision)))
}

func (h *HTTPHandlers) getFraudDecision(w http.ResponseWriter, r *http.Request) {
	customerID, ok := h.resolveCustomerID(w, r)
	if !ok {
		return
	}
	if h.AllowFraudDecision != nil && !h.AllowFraudDecision(customerID.String()) {
		httpresponse.Error(w, http.StatusTooManyRequests, "TOO_MANY_REQUESTS", "fraud decision lookup rate limit exceeded")
		return
	}

	ipHash := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("ip_hash")))
	if !ValidMLIPHashHex(ipHash) {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "ip_hash must be 32 hex characters")
		return
	}

	hours := 24
	if raw := strings.TrimSpace(r.URL.Query().Get("hours")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid hours")
			return
		}
		hours = parsed
	}
	if hours > ExplainMaxHours {
		hours = ExplainMaxHours
	}

	var campaignID *uuid.UUID
	if raw := strings.TrimSpace(r.URL.Query().Get("campaign_id")); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid campaign_id")
			return
		}
		if h.AuthorizeCampaignAccess != nil {
			if err := h.AuthorizeCampaignAccess(r, id); err != nil {
				h.writeServiceError(w, err)
				return
			}
		}
		campaignID = &id
	}

	out, err := h.Decisions.ExplainFraudDecision(r.Context(), customerID, ipHash, campaignID, hours)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, out)
}
