package controlplane

import (
	"net/http"

	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
)

func (h *ReportsHTTPHandlers) resolveReportCustomerID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	var customerID uuid.UUID
	if custIDStr := r.URL.Query().Get("customer_id"); custIDStr != "" {
		id, err := uuid.Parse(custIDStr)
		if err != nil {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid customer_id")
			return uuid.Nil, false
		}
		customerID = id
	} else if h.ResolveForecastCustomerID != nil {
		resolved, err := h.ResolveForecastCustomerID(r, nil)
		if err != nil {
			h.writeServiceError(w, err)
			return uuid.Nil, false
		}
		if resolved != nil {
			customerID = *resolved
		}
	}
	if customerID == uuid.Nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "customer_id required")
		return uuid.Nil, false
	}
	if h.AuthorizeCustomerAccess != nil {
		if err := h.AuthorizeCustomerAccess(r, customerID.String()); err != nil {
			h.writeServiceError(w, err)
			return uuid.Nil, false
		}
	}
	return customerID, true
}

func (h *ReportsHTTPHandlers) authorizeReportCampaign(w http.ResponseWriter, r *http.Request, campaignID uuid.UUID) bool {
	if h.AuthorizeCampaignAccess == nil {
		return true
	}
	if err := h.AuthorizeCampaignAccess(r, campaignID); err != nil {
		h.writeServiceError(w, err)
		return false
	}
	return true
}
