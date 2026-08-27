package controlplane

import (
	"context"
	"net/http"

	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
)

type FraudIntegrationDTO struct {
	CampaignID    string `json:"campaign_id"`
	Name          string `json:"name"`
	Provider      string `json:"provider,omitempty"`
	Configured    bool   `json:"configured"`
	HealthStatus  string `json:"health_status"`
	LastSuccessAt string `json:"last_success_at,omitempty"`
	DLQCount      int64  `json:"dlq_count"`
	LastError     string `json:"last_error,omitempty"`
}

type FraudIntegrationsService interface {
	ListFraudIntegrationsForCustomer(ctx context.Context, customerID uuid.UUID) ([]FraudIntegrationDTO, error)
}

func (h *FraudHTTPHandlers) registerFraudIntegrationRoutes(mux *http.ServeMux, limit func(http.HandlerFunc) http.HandlerFunc, perm func(string, http.HandlerFunc) http.HandlerFunc) {
	if h == nil || h.Integrations == nil {
		return
	}
	mux.HandleFunc("GET /api/v1/fraud/integrations", limit(perm("audit:read", h.listFraudIntegrations)))
}

func (h *FraudHTTPHandlers) listFraudIntegrations(w http.ResponseWriter, r *http.Request) {
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
