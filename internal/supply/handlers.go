package supply

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"
)

type AdminService interface {
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
	ValidateSupplyFiles(ctx context.Context) (ValidationDTO, error)
	SupplyExportPath() string
}

type HTTPHandlers struct {
	Admin                AdminService
	ApplyRateLimit       func(http.HandlerFunc) http.HandlerFunc
	RequirePermission    func(string, http.HandlerFunc) http.HandlerFunc
	RequireAnyPermission func([]string, http.HandlerFunc) http.HandlerFunc
	WriteServiceError    func(http.ResponseWriter, error)
}

func (h *HTTPHandlers) Register(mux *http.ServeMux) {
	if h == nil || h.Admin == nil {
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

func (h *HTTPHandlers) listSellers(w http.ResponseWriter, r *http.Request) {
	rows, err := h.Admin.ListSellers(r.Context())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, rows)
}

func (h *HTTPHandlers) createSeller(w http.ResponseWriter, r *http.Request) {
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}
	var req SellerWriteRequest
	if err := json.Unmarshal(body, &req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json body")
		return
	}
	row, err := h.Admin.CreateSeller(r.Context(), req)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusCreated, row)
}

func (h *HTTPHandlers) updateSeller(w http.ResponseWriter, r *http.Request) {
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
	row, err := h.Admin.UpdateSeller(r.Context(), id, req)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, row)
}

func (h *HTTPHandlers) deleteSeller(w http.ResponseWriter, r *http.Request) {
	id, err := parsePathInt64(r, "id")
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid seller id")
		return
	}
	if err := h.Admin.DeleteSeller(r.Context(), id); err != nil {
		h.writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *HTTPHandlers) listAdsTxt(w http.ResponseWriter, r *http.Request) {
	rows, err := h.Admin.ListAdsTxtEntries(r.Context())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, rows)
}

func (h *HTTPHandlers) createAdsTxt(w http.ResponseWriter, r *http.Request) {
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}
	var req AdsTxtWriteRequest
	if err := json.Unmarshal(body, &req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json body")
		return
	}
	row, err := h.Admin.CreateAdsTxtEntry(r.Context(), req)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusCreated, row)
}

func (h *HTTPHandlers) updateAdsTxt(w http.ResponseWriter, r *http.Request) {
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
	row, err := h.Admin.UpdateAdsTxtEntry(r.Context(), id, req)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, row)
}

func (h *HTTPHandlers) deleteAdsTxt(w http.ResponseWriter, r *http.Request) {
	id, err := parsePathInt64(r, "id")
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid ads.txt id")
		return
	}
	if err := h.Admin.DeleteAdsTxtEntry(r.Context(), id); err != nil {
		h.writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *HTTPHandlers) previewSellersJSON(w http.ResponseWriter, r *http.Request) {
	body, err := h.Admin.BuildSellersJSON(r.Context())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

func (h *HTTPHandlers) previewAdsTxt(w http.ResponseWriter, r *http.Request) {
	body, err := h.Admin.BuildAdsTxt(r.Context())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(body))
}

func (h *HTTPHandlers) getExportPath(w http.ResponseWriter, r *http.Request) {
	httpresponse.JSON(w, http.StatusOK, ExportPathDTO{Path: h.Admin.SupplyExportPath()})
}

func (h *HTTPHandlers) getSupplyValidation(w http.ResponseWriter, r *http.Request) {
	report, err := h.Admin.ValidateSupplyFiles(r.Context())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, report)
}

func (h *HTTPHandlers) writeServiceError(w http.ResponseWriter, err error) {
	if h.WriteServiceError != nil {
		h.WriteServiceError(w, err)
		return
	}
	httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL", "internal error")
}

func parsePathInt64(r *http.Request, key string) (int64, error) {
	raw := r.PathValue(key)
	return strconv.ParseInt(raw, 10, 64)
}
