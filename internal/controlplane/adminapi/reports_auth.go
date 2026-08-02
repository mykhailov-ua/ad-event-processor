package adminapi

import (
	"net/http"

	"espx/pkg/httpresponse"

	"github.com/google/uuid"
)

func (reports *ReportsHTTPHandlers) resolveReportCustomerID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	var customerID uuid.UUID
	if custIDStr := r.URL.Query().Get("customer_id"); custIDStr != "" {
		id, err := uuid.Parse(custIDStr)
		if err != nil {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid customer_id")
			return uuid.Nil, false
		}
		customerID = id
	} else if reports.ResolveForecastCustomerID != nil {
		resolved, err := reports.ResolveForecastCustomerID(r, nil)
		if err != nil {
			reports.writeServiceError(w, err)
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
	if reports.AuthorizeCustomerAccess != nil {
		if err := reports.AuthorizeCustomerAccess(r, customerID.String()); err != nil {
			reports.writeServiceError(w, err)
			return uuid.Nil, false
		}
	}
	return customerID, true
}

func (reports *ReportsHTTPHandlers) authorizeReportCampaign(w http.ResponseWriter, r *http.Request, campaignID uuid.UUID) bool {
	if reports.AuthorizeCampaignAccess == nil {
		return true
	}
	if err := reports.AuthorizeCampaignAccess(r, campaignID); err != nil {
		reports.writeServiceError(w, err)
		return false
	}
	return true
}
