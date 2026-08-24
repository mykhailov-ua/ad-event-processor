package controlplane

import (
	"context"
	"encoding/json"
	"net/http"

	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
)

type BrandDTO struct {
	ID         string `json:"id"`
	CustomerID string `json:"customer_id"`
	Name       string `json:"name"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
	FreqLimit  int32  `json:"freq_limit"`
	FreqWindow int32  `json:"freq_window"`
}

type BrandCreativeDTO struct {
	ID         string `json:"id"`
	BrandID    string `json:"brand_id"`
	Name       string `json:"name"`
	LandingURL string `json:"landing_url"`
	Weight     int32  `json:"weight"`
	Status     string `json:"status"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

func (c BrandCreativeDTO) Scrub(level authz.MaskLevel) BrandCreativeDTO {
	if level == authz.MaskFull {
		return c
	}
	out := c
	out.LandingURL = ""
	return out
}

type SellerDTO struct {
	ID             int64  `json:"id"`
	SellerID       string `json:"seller_id"`
	Domain         string `json:"domain"`
	SellerType     string `json:"seller_type"`
	Name           string `json:"name"`
	IsConfidential bool   `json:"is_confidential"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

type AdsTxtEntryDTO struct {
	ID                 int64  `json:"id"`
	Domain             string `json:"domain"`
	PublisherAccountID string `json:"publisher_account_id"`
	Relationship       string `json:"relationship"`
	CertAuthorityID    string `json:"cert_authority_id,omitempty"`
	SortOrder          int32  `json:"sort_order"`
	CreatedAt          string `json:"created_at"`
	UpdatedAt          string `json:"updated_at"`
}

type CreateBrandRequest struct {
	CustomerID string `json:"customer_id"`
	Name       string `json:"name"`
}

type UpsertBrandCreativeRequest struct {
	Name       string `json:"name"`
	LandingURL string `json:"landing_url"`
	Weight     int32  `json:"weight"`
	Status     string `json:"status"`
}

type UpdateBrandCreativeRequest struct {
	Name       string `json:"name"`
	LandingURL string `json:"landing_url"`
	Weight     int32  `json:"weight"`
	Status     string `json:"status"`
}

type SellerWriteRequest struct {
	SellerID       string `json:"seller_id"`
	Domain         string `json:"domain"`
	SellerType     string `json:"seller_type"`
	Name           string `json:"name"`
	IsConfidential bool   `json:"is_confidential"`
}

type AdsTxtWriteRequest struct {
	Domain             string `json:"domain"`
	PublisherAccountID string `json:"publisher_account_id"`
	Relationship       string `json:"relationship"`
	CertAuthorityID    string `json:"cert_authority_id"`
	SortOrder          int32  `json:"sort_order"`
}

type SupplyExportPathDTO struct {
	Path string `json:"path"`
}

type CommercialAdminService interface {
	ListBrandsByCustomer(ctx context.Context, customerID uuid.UUID) ([]BrandDTO, error)
	CreateBrand(ctx context.Context, customerID uuid.UUID, name string) (uuid.UUID, error)
	ListBrandCreatives(ctx context.Context, brandID uuid.UUID) ([]BrandCreativeDTO, error)
	UpsertBrandCreative(ctx context.Context, brandID uuid.UUID, name, landingURL string, weight int32, status string) (uuid.UUID, error)
	UpdateBrandCreative(ctx context.Context, creativeID uuid.UUID, name, landingURL string, weight int32, status string) error
	DeleteBrandCreative(ctx context.Context, creativeID uuid.UUID) error
	ListSellers(ctx context.Context) ([]SellerDTO, error)
	CreateSeller(ctx context.Context, req SellerWriteRequest) (SellerDTO, error)
	UpdateSeller(ctx context.Context, id int64, req SellerWriteRequest) (SellerDTO, error)
	DeleteSeller(ctx context.Context, id int64) error
	ListAdsTxtEntries(ctx context.Context) ([]AdsTxtEntryDTO, error)
	CreateAdsTxtEntry(ctx context.Context, req AdsTxtWriteRequest) (AdsTxtEntryDTO, error)
	UpdateAdsTxtEntry(ctx context.Context, id int64, req AdsTxtWriteRequest) (AdsTxtEntryDTO, error)
	DeleteAdsTxtEntry(ctx context.Context, id int64) error
	BuildSellersJSON(ctx context.Context) ([]byte, error)
	BuildAdsTxt(ctx context.Context) (string, error)
	ValidateSupplyFiles(ctx context.Context) (SupplyValidationDTO, error)
	SupplyExportPath() string
}

type CommercialHTTPHandlers struct {
	Commercial              CommercialAdminService
	ApplyRateLimit          func(http.HandlerFunc) http.HandlerFunc
	RequirePermission       func(string, http.HandlerFunc) http.HandlerFunc
	RequireAnyPermission    func([]string, http.HandlerFunc) http.HandlerFunc
	AuthorizeCustomerAccess func(*http.Request, string) error
	WriteServiceError       func(http.ResponseWriter, error)
}

func (h *CommercialHTTPHandlers) Register(mux *http.ServeMux) {
	if h == nil || h.Commercial == nil {
		return
	}
	limit := h.ApplyRateLimit
	perm := h.RequirePermission
	permAny := h.RequireAnyPermission
	if limit == nil {
		limit = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	if perm == nil {
		perm = func(_ string, next http.HandlerFunc) http.HandlerFunc { return next }
	}
	if permAny == nil {
		permAny = func(_ []string, next http.HandlerFunc) http.HandlerFunc { return next }
	}

	mux.HandleFunc("GET /api/v1/brands", limit(perm("campaigns:read", h.listBrands)))
	mux.HandleFunc("POST /api/v1/brands", limit(perm("campaigns:write", h.createBrand)))
	mux.HandleFunc("GET /api/v1/brands/{id}/creatives", limit(perm("campaigns:read", h.listBrandCreatives)))
	mux.HandleFunc("POST /api/v1/brands/{id}/creatives", limit(perm("campaigns:write", h.createBrandCreative)))
	mux.HandleFunc("PATCH /api/v1/brand-creatives/{id}", limit(perm("campaigns:write", h.updateBrandCreative)))
	mux.HandleFunc("DELETE /api/v1/brand-creatives/{id}", limit(perm("campaigns:write", h.deleteBrandCreative)))

	mux.HandleFunc("GET /api/v1/supply/sellers", limit(perm("settings:read", h.listSellers)))
	mux.HandleFunc("POST /api/v1/supply/sellers", limit(perm("settings:write", h.createSeller)))
	mux.HandleFunc("PUT /api/v1/supply/sellers/{id}", limit(perm("settings:write", h.updateSeller)))
	mux.HandleFunc("DELETE /api/v1/supply/sellers/{id}", limit(perm("settings:write", h.deleteSeller)))
	mux.HandleFunc("GET /api/v1/supply/ads-txt", limit(perm("settings:read", h.listAdsTxt)))
	mux.HandleFunc("POST /api/v1/supply/ads-txt", limit(perm("settings:write", h.createAdsTxt)))
	mux.HandleFunc("PUT /api/v1/supply/ads-txt/{id}", limit(perm("settings:write", h.updateAdsTxt)))
	mux.HandleFunc("DELETE /api/v1/supply/ads-txt/{id}", limit(perm("settings:write", h.deleteAdsTxt)))
	mux.HandleFunc("GET /api/v1/supply/preview/sellers.json", limit(perm("settings:read", h.previewSellersJSON)))
	mux.HandleFunc("GET /api/v1/supply/preview/ads.txt", limit(perm("settings:read", h.previewAdsTxt)))
	mux.HandleFunc("GET /api/v1/supply/validation", limit(permAny([]string{"settings:read", "supply:read:scoped"}, h.getSupplyValidation)))
	mux.HandleFunc("GET /api/v1/supply/export-path", limit(perm("settings:read", h.getExportPath)))
}

func (h *CommercialHTTPHandlers) listBrands(w http.ResponseWriter, r *http.Request) {
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
	rows, err := h.Commercial.ListBrandsByCustomer(r.Context(), customerID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, rows)
}

func (h *CommercialHTTPHandlers) createBrand(w http.ResponseWriter, r *http.Request) {
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}
	var req CreateBrandRequest
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
	id, err := h.Commercial.CreateBrand(r.Context(), customerID, req.Name)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusCreated, IDCreatedResponse{ID: id.String()})
}

func (h *CommercialHTTPHandlers) listBrandCreatives(w http.ResponseWriter, r *http.Request) {
	brandID, err := coldpath.ParsePathUUID(r, "id")
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid brand id")
		return
	}
	rows, err := h.Commercial.ListBrandCreatives(r.Context(), brandID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, rows)
}

func (h *CommercialHTTPHandlers) createBrandCreative(w http.ResponseWriter, r *http.Request) {
	brandID, err := coldpath.ParsePathUUID(r, "id")
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid brand id")
		return
	}
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}
	var req UpsertBrandCreativeRequest
	if err := json.Unmarshal(body, &req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json body")
		return
	}
	id, err := h.Commercial.UpsertBrandCreative(r.Context(), brandID, req.Name, req.LandingURL, req.Weight, req.Status)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusCreated, IDCreatedResponse{ID: id.String()})
}

func (h *CommercialHTTPHandlers) updateBrandCreative(w http.ResponseWriter, r *http.Request) {
	creativeID, err := coldpath.ParsePathUUID(r, "id")
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid creative id")
		return
	}
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}
	var req UpdateBrandCreativeRequest
	if err := json.Unmarshal(body, &req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json body")
		return
	}
	if err := h.Commercial.UpdateBrandCreative(r.Context(), creativeID, req.Name, req.LandingURL, req.Weight, req.Status); err != nil {
		h.writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *CommercialHTTPHandlers) deleteBrandCreative(w http.ResponseWriter, r *http.Request) {
	creativeID, err := coldpath.ParsePathUUID(r, "id")
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid creative id")
		return
	}
	if err := h.Commercial.DeleteBrandCreative(r.Context(), creativeID); err != nil {
		h.writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *CommercialHTTPHandlers) listSellers(w http.ResponseWriter, r *http.Request) {
	rows, err := h.Commercial.ListSellers(r.Context())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, rows)
}

func (h *CommercialHTTPHandlers) createSeller(w http.ResponseWriter, r *http.Request) {
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}
	var req SellerWriteRequest
	if err := json.Unmarshal(body, &req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json body")
		return
	}
	row, err := h.Commercial.CreateSeller(r.Context(), req)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusCreated, row)
}

func (h *CommercialHTTPHandlers) updateSeller(w http.ResponseWriter, r *http.Request) {
	id, err := parsePathInt64(r, "id")
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid seller id")
		return
	}
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}
	var req SellerWriteRequest
	if err := json.Unmarshal(body, &req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json body")
		return
	}
	row, err := h.Commercial.UpdateSeller(r.Context(), id, req)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, row)
}

func (h *CommercialHTTPHandlers) deleteSeller(w http.ResponseWriter, r *http.Request) {
	id, err := parsePathInt64(r, "id")
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid seller id")
		return
	}
	if err := h.Commercial.DeleteSeller(r.Context(), id); err != nil {
		h.writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *CommercialHTTPHandlers) listAdsTxt(w http.ResponseWriter, r *http.Request) {
	rows, err := h.Commercial.ListAdsTxtEntries(r.Context())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, rows)
}

func (h *CommercialHTTPHandlers) createAdsTxt(w http.ResponseWriter, r *http.Request) {
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}
	var req AdsTxtWriteRequest
	if err := json.Unmarshal(body, &req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json body")
		return
	}
	row, err := h.Commercial.CreateAdsTxtEntry(r.Context(), req)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusCreated, row)
}

func (h *CommercialHTTPHandlers) updateAdsTxt(w http.ResponseWriter, r *http.Request) {
	id, err := parsePathInt64(r, "id")
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid ads.txt id")
		return
	}
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}
	var req AdsTxtWriteRequest
	if err := json.Unmarshal(body, &req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json body")
		return
	}
	row, err := h.Commercial.UpdateAdsTxtEntry(r.Context(), id, req)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, row)
}

func (h *CommercialHTTPHandlers) deleteAdsTxt(w http.ResponseWriter, r *http.Request) {
	id, err := parsePathInt64(r, "id")
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid ads.txt id")
		return
	}
	if err := h.Commercial.DeleteAdsTxtEntry(r.Context(), id); err != nil {
		h.writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *CommercialHTTPHandlers) previewSellersJSON(w http.ResponseWriter, r *http.Request) {
	body, err := h.Commercial.BuildSellersJSON(r.Context())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(body); err != nil {
		return
	}
}

func (h *CommercialHTTPHandlers) previewAdsTxt(w http.ResponseWriter, r *http.Request) {
	body, err := h.Commercial.BuildAdsTxt(r.Context())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(body)); err != nil {
		return
	}
}

func (h *CommercialHTTPHandlers) getExportPath(w http.ResponseWriter, r *http.Request) {
	httpresponse.JSON(w, http.StatusOK, SupplyExportPathDTO{Path: h.Commercial.SupplyExportPath()})
}

func (h *CommercialHTTPHandlers) getSupplyValidation(w http.ResponseWriter, r *http.Request) {
	report, err := h.Commercial.ValidateSupplyFiles(r.Context())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, report)
}

func (h *CommercialHTTPHandlers) writeServiceError(w http.ResponseWriter, err error) {
	if h.WriteServiceError != nil {
		h.WriteServiceError(w, err)
		return
	}
	httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL", "internal error")
}
