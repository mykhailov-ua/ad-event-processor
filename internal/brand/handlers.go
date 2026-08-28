package brand

import (
	"context"
	"encoding/json"
	"net/http"

	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
)

type AdminService interface {
	ListBrandsByCustomer(ctx context.Context, customerID uuid.UUID) ([]DTO, error)
	CreateBrand(ctx context.Context, customerID uuid.UUID, name string) (uuid.UUID, error)
	ListBrandCreatives(ctx context.Context, brandID uuid.UUID) ([]CreativeDTO, error)
	UpsertBrandCreative(ctx context.Context, brandID uuid.UUID, name, landingURL string, weight int32, status string) (uuid.UUID, error)
	UpdateBrandCreative(ctx context.Context, creativeID uuid.UUID, name, landingURL string, weight int32, status string) error
	DeleteBrandCreative(ctx context.Context, creativeID uuid.UUID) error
}

type HTTPHandlers struct {
	Admin                   AdminService
	ApplyRateLimit          func(http.HandlerFunc) http.HandlerFunc
	RequirePermission       func(string, http.HandlerFunc) http.HandlerFunc
	AuthorizeCustomerAccess func(*http.Request, string) error
	WriteServiceError       func(http.ResponseWriter, error)
}

func (h *HTTPHandlers) Register(mux *http.ServeMux) {
	if h == nil || h.Admin == nil {
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

	mux.HandleFunc("GET /api/v1/brands", limit(perm("campaigns:read", h.listBrands)))
	mux.HandleFunc("POST /api/v1/brands", limit(perm("campaigns:write", h.createBrand)))
	mux.HandleFunc("GET /api/v1/brands/{id}/creatives", limit(perm("campaigns:read", h.listBrandCreatives)))
	mux.HandleFunc("POST /api/v1/brands/{id}/creatives", limit(perm("campaigns:write", h.createBrandCreative)))
	mux.HandleFunc("PATCH /api/v1/brand-creatives/{id}", limit(perm("campaigns:write", h.updateBrandCreative)))
	mux.HandleFunc("DELETE /api/v1/brand-creatives/{id}", limit(perm("campaigns:write", h.deleteBrandCreative)))
}

func (h *HTTPHandlers) listBrands(w http.ResponseWriter, r *http.Request) {
	custStr := r.URL.Query().Get("customer_id")
	if custStr == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "customer_id required")
		return
	}
	customerID, err := uuid.Parse(custStr)
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
	rows, err := h.Admin.ListBrandsByCustomer(r.Context(), customerID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, rows)
}

func (h *HTTPHandlers) createBrand(w http.ResponseWriter, r *http.Request) {
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}
	var req CreateRequest
	if err := json.Unmarshal(body, &req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json body")
		return
	}
	customerID, err := uuid.Parse(req.CustomerID)
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
	id, err := h.Admin.CreateBrand(r.Context(), customerID, req.Name)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusCreated, createdIDResponse{ID: id.String()})
}

func (h *HTTPHandlers) listBrandCreatives(w http.ResponseWriter, r *http.Request) {
	brandID, err := coldpath.ParsePathUUID(r, "id")
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid brand id")
		return
	}
	rows, err := h.Admin.ListBrandCreatives(r.Context(), brandID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, rows)
}

func (h *HTTPHandlers) createBrandCreative(w http.ResponseWriter, r *http.Request) {
	brandID, err := coldpath.ParsePathUUID(r, "id")
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid brand id")
		return
	}
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}
	var req UpsertCreativeRequest
	if err := json.Unmarshal(body, &req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json body")
		return
	}
	id, err := h.Admin.UpsertBrandCreative(r.Context(), brandID, req.Name, req.LandingURL, req.Weight, req.Status)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusCreated, createdIDResponse{ID: id.String()})
}

func (h *HTTPHandlers) updateBrandCreative(w http.ResponseWriter, r *http.Request) {
	creativeID, err := coldpath.ParsePathUUID(r, "id")
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid creative id")
		return
	}
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}
	var req UpdateCreativeRequest
	if err := json.Unmarshal(body, &req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json body")
		return
	}
	if err := h.Admin.UpdateBrandCreative(r.Context(), creativeID, req.Name, req.LandingURL, req.Weight, req.Status); err != nil {
		h.writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *HTTPHandlers) deleteBrandCreative(w http.ResponseWriter, r *http.Request) {
	creativeID, err := coldpath.ParsePathUUID(r, "id")
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid creative id")
		return
	}
	if err := h.Admin.DeleteBrandCreative(r.Context(), creativeID); err != nil {
		h.writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *HTTPHandlers) writeServiceError(w http.ResponseWriter, err error) {
	if h.WriteServiceError != nil {
		h.WriteServiceError(w, err)
		return
	}
	httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL", "internal error")
}
