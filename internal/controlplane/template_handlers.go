package controlplane

import (
	"context"
	"net/http"

	"ad-event-processor/internal/integrationschema"
	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
)

type TemplateCatalogService interface {
	ListBundledTemplates(ctx context.Context) []integrationschema.TemplateCatalogEntry
	ImportBundledTemplates(ctx context.Context, names []string) ([]IntegrationSchemaDTO, error)
	ApplyCampaignTemplates(ctx context.Context, campaignID uuid.UUID, req ApplyCampaignTemplatesRequest) (ApplyCampaignTemplatesResult, error)
}

type ImportTemplatesRequest struct {
	Names []string `json:"names,omitempty"`
}

type ApplyCampaignTemplatesRequest struct {
	TrafficSource    string `json:"traffic_source,omitempty"`
	AffiliateNetwork string `json:"affiliate_network,omitempty"`
	TrackingDomain   string `json:"tracking_domain,omitempty"`
}

type ApplyCampaignTemplatesResult struct {
	CampaignID        string            `json:"campaign_id"`
	TrafficSource     map[string]string `json:"traffic_source,omitempty"`
	AffiliatePostback map[string]string `json:"affiliate_postback,omitempty"`
	AffiliateStatus   map[string]string `json:"affiliate_status,omitempty"`
}

func (h *IntegrationSchemaHTTPHandlers) RegisterTemplateRoutes(mux *http.ServeMux) {
	if h == nil {
		return
	}
	svc, ok := h.TemplateCatalog.(TemplateCatalogService)
	if !ok || svc == nil {
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
	mux.HandleFunc("GET /api/v1/integration/templates", limit(perm("campaigns:read", h.listBundledTemplates)))
	mux.HandleFunc("POST /api/v1/integration/templates/import", limit(perm("campaigns:write", h.importBundledTemplates)))
	mux.HandleFunc("POST /api/v1/campaigns/{id}/apply-templates", limit(perm("campaigns:write", h.applyCampaignTemplates)))
}

func (h *IntegrationSchemaHTTPHandlers) listBundledTemplates(w http.ResponseWriter, r *http.Request) {
	svc, ok := h.TemplateCatalog.(TemplateCatalogService)
	if !ok || svc == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "template catalog unavailable")
		return
	}
	out := svc.ListBundledTemplates(r.Context())
	if out == nil {
		out = []integrationschema.TemplateCatalogEntry{}
	}
	httpresponse.JSON(w, http.StatusOK, out)
}

func (h *IntegrationSchemaHTTPHandlers) importBundledTemplates(w http.ResponseWriter, r *http.Request) {
	svc, ok := h.TemplateCatalog.(TemplateCatalogService)
	if !ok || svc == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "template catalog unavailable")
		return
	}
	req, ok := coldpath.DecodeRequestOrBadRequest[ImportTemplatesRequest](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	imported, err := svc.ImportBundledTemplates(r.Context(), req.Names)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	if imported == nil {
		imported = []IntegrationSchemaDTO{}
	}
	httpresponse.JSON(w, http.StatusOK, imported)
}

func (h *IntegrationSchemaHTTPHandlers) applyCampaignTemplates(w http.ResponseWriter, r *http.Request) {
	svc, ok := h.TemplateCatalog.(TemplateCatalogService)
	if !ok || svc == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "template catalog unavailable")
		return
	}
	campaignID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid campaign id")
		return
	}
	req, ok := coldpath.DecodeRequestOrBadRequest[ApplyCampaignTemplatesRequest](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	result, err := svc.ApplyCampaignTemplates(r.Context(), campaignID, req)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, result)
}
