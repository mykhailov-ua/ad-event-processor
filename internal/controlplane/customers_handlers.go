package controlplane

import (
	"context"
	"net/http"

	"github.com/bidshard/ad-event-processor/pkg/coldpath"
	"github.com/bidshard/ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
)

type CustomerDTO struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Balance         string `json:"balance"`
	Currency        string `json:"currency"`
	ActiveCampaigns int64  `json:"active_campaigns"`
	TotalSpend      string `json:"total_spend"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

type CustomerListResponse = ListResponse[CustomerDTO]

type CustomerReader interface {
	ListCustomers(ctx context.Context, limit, offset int32) ([]CustomerDTO, int64, error)
	GetCustomerDTO(ctx context.Context, id uuid.UUID) (CustomerDTO, error)
}

type CustomersHTTPHandlers struct {
	Customers               CustomerReader
	ApplyRateLimit          func(http.HandlerFunc) http.HandlerFunc
	RequirePermission       func(string, http.HandlerFunc) http.HandlerFunc
	AuthorizeCustomerAccess func(*http.Request, string) error
	WriteServiceError       func(http.ResponseWriter, error)
}

func (h *CustomersHTTPHandlers) Register(mux *http.ServeMux) {
	if h == nil || h.Customers == nil {
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
	mux.HandleFunc("GET /api/v1/customers", limit(perm("customers:read", h.listCustomers)))
	mux.HandleFunc("GET /api/v1/customers/{id}", limit(perm("customers:read", h.getCustomer)))
}

func (h *CustomersHTTPHandlers) listCustomers(w http.ResponseWriter, r *http.Request) {
	limit, offset := coldpath.ParseAPIPagination(r)
	items, total, err := h.Customers.ListCustomers(r.Context(), limit, offset)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, CustomerListResponse{Items: items, Total: total})
}

func (h *CustomersHTTPHandlers) getCustomer(w http.ResponseWriter, r *http.Request) {
	rawID := r.PathValue("id")
	customerID, err := uuid.Parse(rawID)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid customer_id")
		return
	}
	if h.AuthorizeCustomerAccess != nil {
		if err := h.AuthorizeCustomerAccess(r, customerID.String()); err != nil {
			h.writeServiceError(w, err)
			return
		}
	}
	cust, err := h.Customers.GetCustomerDTO(r.Context(), customerID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, cust)
}

func (h *CustomersHTTPHandlers) writeServiceError(w http.ResponseWriter, err error) {
	if h.WriteServiceError != nil {
		h.WriteServiceError(w, err)
		return
	}
	httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal error")
}
