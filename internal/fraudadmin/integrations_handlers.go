package fraudadmin

import (
	"net/http"

	"ad-event-processor/pkg/httpresponse"
)

func (h *HTTPHandlers) registerFraudIntegrationRoutes(mux *http.ServeMux, limit func(http.HandlerFunc) http.HandlerFunc, perm func(string, http.HandlerFunc) http.HandlerFunc) {
	if h == nil || h.Integrations == nil {
		return
	}
	mux.HandleFunc("GET /api/v1/fraud/integrations", limit(perm("audit:read", h.listFraudIntegrations)))
}

func (h *HTTPHandlers) listFraudIntegrations(w http.ResponseWriter, r *http.Request) {
	customerID, ok := h.resolveCustomerID(w, r)
	if !ok {
		return
	}
	rows, err := h.Integrations.ListFraudIntegrationsForCustomer(r.Context(), customerID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	if rows == nil {
		rows = []FraudIntegrationDTO{}
	}
	httpresponse.JSON(w, http.StatusOK, rows)
}
