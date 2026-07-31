package adminapi

import (
	"context"
	"net/http"
	"time"

	"espx/internal/config"
	"espx/pkg/branding"
	"espx/pkg/httpresponse"
)

type MetaLicenseDTO struct {
	State          string `json:"state"`
	ValidUntil     string `json:"valid_until,omitempty"`
	BannerSeverity string `json:"banner_severity,omitempty"`
	RenewDays      int    `json:"renew_days,omitempty"`
}

type MetaResponseDTO struct {
	ProductName    string          `json:"product_name"`
	VendorName     string          `json:"vendor_name"`
	SiteURL        string          `json:"site_url"`
	Version        string          `json:"version"`
	IngressSchemas []string        `json:"ingress_schemas"`
	DeploymentID   string          `json:"deployment_id,omitempty"`
	SKU            string          `json:"sku,omitempty"`
	License        *MetaLicenseDTO `json:"license,omitempty"`
}

type MetaEnricher func(ctx context.Context) (deploymentID, sku string, license *MetaLicenseDTO, err error)

type MetaHTTPHandlers struct {
	ApplyRateLimit func(http.HandlerFunc) http.HandlerFunc
	Enrich         MetaEnricher
	WriteError     func(http.ResponseWriter, error)
}

func (h *MetaHTTPHandlers) Register(mux *http.ServeMux) {
	if h == nil || mux == nil {
		return
	}
	limit := h.ApplyRateLimit
	if limit == nil {
		limit = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	mux.HandleFunc("GET /api/v1/meta", limit(h.getMeta))
}

func (h *MetaHTTPHandlers) getMeta(w http.ResponseWriter, r *http.Request) {
	resp := MetaResponseDTO{
		ProductName:    branding.ProductName(),
		VendorName:     branding.VendorName(),
		SiteURL:        branding.SiteURL(),
		Version:        branding.Version(),
		IngressSchemas: config.SupportedIngressSchemas(),
	}
	if h.Enrich != nil {
		deploymentID, sku, license, err := h.Enrich(r.Context())
		if err != nil {
			if h.WriteError != nil {
				h.WriteError(w, err)
				return
			}
			httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal error")
			return
		}
		resp.DeploymentID = deploymentID
		resp.SKU = sku
		resp.License = license
	}
	httpresponse.JSON(w, http.StatusOK, resp)
}

func BuildMetaLicense(state, bannerSeverity string, validUntil time.Time, hasValidUntil bool) *MetaLicenseDTO {
	if state == "" {
		return nil
	}
	lic := &MetaLicenseDTO{
		State:          state,
		BannerSeverity: bannerSeverity,
	}
	if hasValidUntil {
		lic.ValidUntil = validUntil.Format(time.RFC3339)
		days := int(time.Until(validUntil).Hours() / 24)
		if days >= 0 {
			lic.RenewDays = days
		}
	}
	return lic
}
