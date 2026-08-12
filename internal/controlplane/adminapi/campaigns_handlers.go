package adminapi

import (
	"context"
	"net/http"

	"github.com/bidshard/ad-event-processor/pkg/coldpath"
	"github.com/bidshard/ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
)

type CampaignListResponse struct {
	Items []CampaignDTO `json:"items"`
	Total int64         `json:"total"`
}

type CampaignReader interface {
	GetCampaign(ctx context.Context, campaignID uuid.UUID) (CampaignDTO, error)
	GetCampaignMargin(ctx context.Context, campaignID uuid.UUID) (CampaignMarginDTO, error)
	ListCampaigns(ctx context.Context, customerID uuid.UUID, status string, limit, offset int32) ([]CampaignDTO, int64, error)
	AttachCampaignListMarginBreach(ctx context.Context, items []CampaignDTO)
	PatchCampaign(ctx context.Context, campaignID uuid.UUID, req PatchCampaignRequest) (CampaignDTO, error)
	ListCampaignEvents(ctx context.Context, campaignID uuid.UUID, limit, offset int32) ([]CampaignEventDTO, int64, error)
}

type CampaignsHTTPHandlers struct {
	Campaigns               CampaignReader
	ApplyRateLimit          func(http.HandlerFunc) http.HandlerFunc
	RequireAnyPermission    func([]string, http.HandlerFunc) http.HandlerFunc
	AuthorizeCampaignAccess func(*http.Request, uuid.UUID) error
	ResolveCustomerID       func(*http.Request, *uuid.UUID) (uuid.UUID, error)
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
	mux.HandleFunc("GET /api/v1/campaigns/{id}/events", limit(perm([]string{"campaigns:read"}, campaigns.listCampaignEvents)))
	mux.HandleFunc("GET /api/v1/campaigns/{id}/margin", limit(perm([]string{"campaigns:read"}, campaigns.getCampaignMargin)))
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
	limit, offset := parseAPIPagination(r)

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

func (campaigns *CampaignsHTTPHandlers) listCampaignEvents(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := campaigns.parseCampaignID(w, r)
	if !ok {
		return
	}
	limit, offset := parseAPIPagination(r)
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
