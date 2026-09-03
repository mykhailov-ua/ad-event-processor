package campaign

import (
	"context"
	"net/http"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type CampaignTargetCountriesResponse struct {
	Countries []string `json:"countries"`
}

func (h *CampaignsHTTPHandlers) listCampaignTargetCountries(w http.ResponseWriter, r *http.Request) {
	if h.PostgresPool == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "postgres unavailable")
		return
	}

	var customerID uuid.UUID
	if custStr := r.URL.Query().Get("customer_id"); custStr != "" {
		id, err := uuid.Parse(custStr)
		if err != nil {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid customer_id")
			return
		}
		customerID = id
	}
	if h.ResolveCustomerID != nil {
		resolved, err := h.ResolveCustomerID(r, nonNilUUID(customerID))
		if err != nil {
			h.WriteHandlerError(w, err)
			return
		}
		customerID = resolved
	}

	var customerArg pgtype.UUID
	if customerID != uuid.Nil {
		customerArg = domain.ToUUID(customerID)
	}

	q := db.New(h.PostgresPool)
	countries, err := q.ListCampaignTargetCountries(r.Context(), customerArg)
	if err != nil {
		h.WriteHandlerError(w, err)
		return
	}
	if countries == nil {
		countries = []string{}
	}

	httpresponse.JSON(w, http.StatusOK, CampaignTargetCountriesResponse{Countries: countries})
}

func QueryCampaignTargetCountries(ctx context.Context, q *db.Queries, customerID uuid.UUID) ([]string, error) {
	var customerArg pgtype.UUID
	if customerID != uuid.Nil {
		customerArg = domain.ToUUID(customerID)
	}
	countries, err := q.ListCampaignTargetCountries(ctx, customerArg)
	if err != nil {
		return nil, err
	}
	if countries == nil {
		return []string{}, nil
	}
	return countries, nil
}
