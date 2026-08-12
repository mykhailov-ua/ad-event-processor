package adminapi

import (
	"context"
	"net/http"
	"time"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/pkg/branding"
	"github.com/bidshard/ad-event-processor/pkg/httpresponse"
)

type MetaLicenseDTO struct {
	State          string   `json:"state"`
	PlanCode       string   `json:"plan_code,omitempty"`
	ValidUntil     string   `json:"valid_until,omitempty"`
	BannerSeverity string   `json:"banner_severity,omitempty"`
	RenewDays      int      `json:"renew_days,omitempty"`
	TierWarnings   []string `json:"tier_warnings,omitempty"`
}

type MetaLicenseBuildInput struct {
	State          string
	BannerSeverity string
	PlanCode       string
	ValidUntil     time.Time
	HasValidUntil  bool
	TierWarnings   []string
}

type MetaResponseDTO struct {
	ProductName       string          `json:"product_name"`
	VendorName        string          `json:"vendor_name"`
	SiteURL           string          `json:"site_url"`
	Version           string          `json:"version"`
	IngressSchemas    []string        `json:"ingress_schemas"`
	DeploymentID      string          `json:"deployment_id,omitempty"`
	BootstrapComplete bool            `json:"bootstrap_complete"`
	EulaVersion       string          `json:"eula_version,omitempty"`
	EulaAccepted      bool            `json:"eula_accepted"`
	EulaRequired      bool            `json:"eula_required"`
	License           *MetaLicenseDTO `json:"license,omitempty"`
}

type MetaEnrichOut struct {
	DeploymentID      string
	License           *MetaLicenseDTO
	BootstrapComplete bool
	EulaVersion       string
	EulaAccepted      bool
	EulaRequired      bool
}

type MetaEnricher func(ctx context.Context) (MetaEnrichOut, error)

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
		out, err := h.Enrich(r.Context())
		if err != nil {
			if h.WriteError != nil {
				h.WriteError(w, err)
				return
			}
			httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal error")
			return
		}
		resp.DeploymentID = out.DeploymentID
		resp.License = out.License
		resp.BootstrapComplete = out.BootstrapComplete
		resp.EulaVersion = out.EulaVersion
		resp.EulaAccepted = out.EulaAccepted
		resp.EulaRequired = out.EulaRequired
	}
	httpresponse.JSON(w, http.StatusOK, resp)
}

func BuildMetaLicense(in MetaLicenseBuildInput) *MetaLicenseDTO {
	if in.State == "" {
		return nil
	}
	lic := &MetaLicenseDTO{
		State:          in.State,
		PlanCode:       in.PlanCode,
		BannerSeverity: in.BannerSeverity,
		TierWarnings:   in.TierWarnings,
	}
	if in.HasValidUntil {
		lic.ValidUntil = in.ValidUntil.Format(time.RFC3339)
		days := int(time.Until(in.ValidUntil).Hours() / 24)
		if days >= 0 {
			lic.RenewDays = days
		}
	}
	return lic
}
