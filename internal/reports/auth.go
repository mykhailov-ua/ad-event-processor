package reports

import (
	"errors"
	"net/http"

	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
)

var (
	errReportCustomerIDRequired = errors.New("report customer_id required")
	errInvalidReportCustomerID  = errors.New("invalid customer_id")
	errClickHouseUnavailable    = errors.New("clickhouse unavailable")
	errInvalidReportCursor      = errors.New("invalid report cursor")
)

func (h *ReportsHTTPHandlers) parseReportCustomerID(r *http.Request) (uuid.UUID, error) {
	var customerID uuid.UUID
	if custIDStr := r.URL.Query().Get("customer_id"); custIDStr != "" {
		id, err := uuid.Parse(custIDStr)
		if err != nil {
			return uuid.Nil, errInvalidReportCustomerID
		}
		customerID = id
	} else if h.ResolveForecastCustomerID != nil {
		resolved, err := h.ResolveForecastCustomerID(r, nil)
		if err != nil {
			return uuid.Nil, err
		}
		if resolved != nil {
			customerID = *resolved
		}
	}
	if customerID == uuid.Nil {
		return uuid.Nil, errReportCustomerIDRequired
	}
	if h.AuthorizeCustomerAccess != nil {
		if err := h.AuthorizeCustomerAccess(r, customerID.String()); err != nil {
			return uuid.Nil, err
		}
	}
	return customerID, nil
}

func (h *ReportsHTTPHandlers) resolveReportCustomerID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	customerID, err := h.parseReportCustomerID(r)
	if err != nil {
		switch {
		case errors.Is(err, errInvalidReportCustomerID), errors.Is(err, errReportCustomerIDRequired):
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		default:
			h.writeServiceError(w, err)
		}
		return uuid.Nil, false
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
