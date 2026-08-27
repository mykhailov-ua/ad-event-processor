package controlplane

import (
	"context"
	"net/http"
	"strings"

	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
)

type CampaignListResponse = ListResponse[CampaignDTO]

type CampaignReader interface {
	GetCampaign(ctx context.Context, campaignID uuid.UUID) (CampaignDTO, error)
	GetCampaignMargin(ctx context.Context, campaignID uuid.UUID) (CampaignMarginDTO, error)
	ListCampaigns(ctx context.Context, customerID uuid.UUID, status string, limit, offset int32) ([]CampaignDTO, int64, error)
	AttachCampaignListMarginBreach(ctx context.Context, items []CampaignDTO)
	PatchCampaign(ctx context.Context, campaignID uuid.UUID, req PatchCampaignRequest) (CampaignDTO, error)
	AssignCampaignOwner(ctx context.Context, campaignID, ownerUserID uuid.UUID) error
	ListCampaignEvents(ctx context.Context, campaignID uuid.UUID, limit, offset int32) ([]CampaignEventDTO, int64, error)
	BlockCampaignPlacement(ctx context.Context, campaignID uuid.UUID, placementID string) error
	CloneCampaign(ctx context.Context, spec CloneCampaignSpec) (CloneCampaignResult, error)
	ExportCampaign(ctx context.Context, campaignID uuid.UUID) (CampaignExportBundle, error)
	ImportCampaign(ctx context.Context, spec ImportCampaignSpec) (ImportCampaignResult, error)
	ImportMigrationCampaigns(ctx context.Context, spec ImportMigrationSpec) (ImportMigrationResult, error)
	GetCampaignIntegrationHealth(ctx context.Context, campaignID uuid.UUID) (IntegrationHealthDTO, error)
}

type CampaignsHTTPHandlers struct {
	Campaigns               CampaignReader
	CampaignFraud           CampaignFraudService
	ConversionMappings      ConversionMappingService
	ApplyRateLimit          func(http.HandlerFunc) http.HandlerFunc
	RequireAnyPermission    func([]string, http.HandlerFunc) http.HandlerFunc
	AuthorizeCampaignAccess func(*http.Request, uuid.UUID) error
	ResolveCustomerID       func(*http.Request, *uuid.UUID) (uuid.UUID, error)
	AllowFraudPreview       func(campaignID string) bool
	WriteServiceError       func(http.ResponseWriter, error)
}

func (campaigns *CampaignsHTTPHandlers) Register(mux *http.ServeMux) {
	if campaigns == nil || campaigns.Campaigns == nil {
		return
	}
	limit := campaigns.ApplyRateLimit
	perm := campaigns.RequireAnyPermission
	if limit == nil {
		limit = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	if perm == nil {
		perm = func(_ []string, next http.HandlerFunc) http.HandlerFunc { return next }
	}
	mux.HandleFunc("GET /api/v1/campaigns", limit(perm([]string{"campaigns:read", "campaigns:read:masked"}, campaigns.listCampaigns)))
	mux.HandleFunc("GET /api/v1/campaigns/{id}", limit(perm([]string{"campaigns:read", "campaigns:read:masked"}, campaigns.getCampaign)))
	mux.HandleFunc("PATCH /api/v1/campaigns/{id}", limit(perm([]string{"campaigns:write"}, campaigns.patchCampaign)))
	mux.HandleFunc("PUT /api/v1/campaigns/{id}/owner", limit(perm([]string{"campaigns:write"}, campaigns.assignCampaignOwner)))
	mux.HandleFunc("GET /api/v1/campaigns/{id}/events", limit(perm([]string{"campaigns:read"}, campaigns.listCampaignEvents)))
	mux.HandleFunc("GET /api/v1/campaigns/{id}/margin", limit(perm([]string{"campaigns:read"}, campaigns.getCampaignMargin)))
	mux.HandleFunc("POST /api/v1/campaigns/{id}/placement-blocks", limit(perm([]string{"campaigns:write"}, campaigns.blockCampaignPlacement)))
	mux.HandleFunc("POST /api/v1/campaigns/{id}/clone", limit(perm([]string{"campaigns:write"}, campaigns.cloneCampaign)))
	mux.HandleFunc("GET /api/v1/campaigns/{id}/export", limit(perm([]string{"campaigns:read"}, campaigns.exportCampaign)))
	mux.HandleFunc("POST /api/v1/campaigns/import", limit(perm([]string{"campaigns:write"}, campaigns.importCampaign)))
	campaigns.registerMigrationRoutes(mux, limit, perm)
	campaigns.registerIntegrationHealthRoutes(mux, limit, perm)
	campaigns.registerConversionMappingRoutes(mux, limit, perm)
	campaigns.registerCampaignFraudRoutes(mux, limit, perm)
}

func (campaigns *CampaignsHTTPHandlers) listCampaigns(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	var customerID uuid.UUID
	if custStr := q.Get("customer_id"); custStr != "" {
		id, err := uuid.Parse(custStr)
		if err != nil {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid customer_id")
			return
		}
		customerID = id
	}

	if campaigns.ResolveCustomerID != nil {
		resolved, err := campaigns.ResolveCustomerID(r, nonNilUUID(customerID))
		if err != nil {
			campaigns.writeServiceError(w, err)
			return
		}
		customerID = resolved
	}

	status := q.Get("status")
	limit, offset := coldpath.ParseAPIPagination(r)

	items, total, err := campaigns.Campaigns.ListCampaigns(r.Context(), customerID, status, limit, offset)
	if err != nil {
		campaigns.writeServiceError(w, err)
		return
	}
	campaigns.Campaigns.AttachCampaignListMarginBreach(r.Context(), items)
	httpresponse.JSON(w, http.StatusOK, CampaignListResponse{Items: items, Total: total})
}

func (campaigns *CampaignsHTTPHandlers) getCampaign(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := campaigns.parseCampaignID(w, r)
	if !ok {
		return
	}
	campaign, err := campaigns.Campaigns.GetCampaign(r.Context(), campaignID)
	if err != nil {
		campaigns.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, campaign)
}

func (campaigns *CampaignsHTTPHandlers) getCampaignMargin(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := campaigns.parseCampaignID(w, r)
	if !ok {
		return
	}
	margin, err := campaigns.Campaigns.GetCampaignMargin(r.Context(), campaignID)
	if err != nil {
		campaigns.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, margin)
}

func (campaigns *CampaignsHTTPHandlers) patchCampaign(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := campaigns.parseCampaignID(w, r)
	if !ok {
		return
	}
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}
	req, err := coldpath.DecodeBody[PatchCampaignRequest](body)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	updated, err := campaigns.Campaigns.PatchCampaign(r.Context(), campaignID, req)
	if err != nil {
		campaigns.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, updated)
}

func (campaigns *CampaignsHTTPHandlers) assignCampaignOwner(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := campaigns.parseCampaignID(w, r)
	if !ok {
		return
	}
	req, ok := coldpath.DecodeRequestOrBadRequest[AssignCampaignOwnerRequest](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	ownerID, err := uuid.Parse(strings.TrimSpace(req.UserID))
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid user_id")
		return
	}
	if err := campaigns.Campaigns.AssignCampaignOwner(r.Context(), campaignID, ownerID); err != nil {
		campaigns.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (campaigns *CampaignsHTTPHandlers) blockCampaignPlacement(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := campaigns.parseCampaignID(w, r)
	if !ok {
		return
	}
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}
	req, err := coldpath.DecodeBody[struct {
		PlacementID string `json:"placement_id"`
	}](body)
	if err != nil || strings.TrimSpace(req.PlacementID) == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if err := campaigns.Campaigns.BlockCampaignPlacement(r.Context(), campaignID, req.PlacementID); err != nil {
		campaigns.writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

type CloneCampaignHTTPRequest struct {
	NamePrefix string `json:"name_prefix,omitempty"`
	NameSuffix string `json:"name_suffix,omitempty"`
}

func (campaigns *CampaignsHTTPHandlers) exportCampaign(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := campaigns.parseCampaignID(w, r)
	if !ok {
		return
	}
	bundle, err := campaigns.Campaigns.ExportCampaign(r.Context(), campaignID)
	if err != nil {
		campaigns.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, bundle)
}

type ImportCampaignHTTPRequest struct {
	CustomerID       string `json:"customer_id"`
	NameOverride     string `json:"name_override,omitempty"`
	BudgetLimitMicro *int64 `json:"budget_limit_micro,omitempty"`
	CampaignExportBundle
}

func (campaigns *CampaignsHTTPHandlers) importCampaign(w http.ResponseWriter, r *http.Request) {
	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idempotencyKey == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Idempotency-Key header is required")
		return
	}
	req, ok := coldpath.DecodeRequestOrBadRequest[ImportCampaignHTTPRequest](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	customerID, err := uuid.Parse(strings.TrimSpace(req.CustomerID))
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid customer_id")
		return
	}
	if campaigns.ResolveCustomerID != nil {
		customerID, err = campaigns.ResolveCustomerID(r, nonNilUUID(customerID))
		if err != nil {
			campaigns.writeServiceError(w, err)
			return
		}
	}
	result, err := campaigns.Campaigns.ImportCampaign(r.Context(), ImportCampaignSpec{
		CustomerID:     customerID,
		NameOverride:   req.NameOverride,
		BudgetOverride: req.BudgetLimitMicro,
		IdempotencyKey: idempotencyKey,
		Bundle:         req.CampaignExportBundle,
	})
	if err != nil {
		campaigns.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusCreated, result)
}

func (campaigns *CampaignsHTTPHandlers) cloneCampaign(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := campaigns.parseCampaignID(w, r)
	if !ok {
		return
	}
	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idempotencyKey == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Idempotency-Key header is required")
		return
	}
	req, ok := coldpath.DecodeRequestOrBadRequest[CloneCampaignHTTPRequest](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	result, err := campaigns.Campaigns.CloneCampaign(r.Context(), CloneCampaignSpec{
		SourceID:       campaignID,
		NamePrefix:     req.NamePrefix,
		NameSuffix:     req.NameSuffix,
		IdempotencyKey: idempotencyKey,
	})
	if err != nil {
		campaigns.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusCreated, result)
}

func (campaigns *CampaignsHTTPHandlers) listCampaignEvents(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := campaigns.parseCampaignID(w, r)
	if !ok {
		return
	}
	limit, offset := coldpath.ParseAPIPagination(r)
	items, total, err := campaigns.Campaigns.ListCampaignEvents(r.Context(), campaignID, limit, offset)
	if err != nil {
		campaigns.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, CampaignEventListResponse{Items: items, Total: total})
}

func (campaigns *CampaignsHTTPHandlers) parseCampaignID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	campaignID, err := coldpath.ParsePathUUID(r, "id")
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid campaign id")
		return uuid.Nil, false
	}
	if campaigns.AuthorizeCampaignAccess != nil {
		if err := campaigns.AuthorizeCampaignAccess(r, campaignID); err != nil {
			campaigns.writeServiceError(w, err)
			return uuid.Nil, false
		}
	}
	return campaignID, true
}

func (campaigns *CampaignsHTTPHandlers) writeServiceError(w http.ResponseWriter, err error) {
	if campaigns.WriteServiceError != nil {
		campaigns.WriteServiceError(w, err)
		return
	}
	httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL", "internal error")
}

func nonNilUUID(id uuid.UUID) *uuid.UUID {
	if id == uuid.Nil {
		return nil
	}
	return &id
}
