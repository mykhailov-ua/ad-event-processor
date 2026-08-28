package campaign

import (
	"context"
	"net/http"

	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
)

type ConversionMappingService interface {
	ListCampaignConversionMappings(ctx context.Context, campaignID uuid.UUID) ([]ConversionMappingDTO, error)
	ReplaceCampaignConversionMappings(ctx context.Context, campaignID uuid.UUID, mappings []ConversionMappingDTO) ([]ConversionMappingDTO, error)
}

func (h *CampaignsHTTPHandlers) registerConversionMappingRoutes(mux *http.ServeMux, limit func(http.HandlerFunc) http.HandlerFunc, perm func([]string, http.HandlerFunc) http.HandlerFunc) {
	if h == nil || h.ConversionMappings == nil {
		return
	}
	read := []string{"campaigns:read", "campaigns:read:masked"}
	mux.HandleFunc("GET /api/v1/campaigns/{id}/conversion-mappings", limit(perm(read, h.getConversionMappings)))
	mux.HandleFunc("PUT /api/v1/campaigns/{id}/conversion-mappings", limit(perm([]string{"campaigns:write"}, h.putConversionMappings)))
}

func (h *CampaignsHTTPHandlers) getConversionMappings(w http.ResponseWriter, r *http.Request) {
	campaignID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid campaign id")
		return
	}
	if h.AuthorizeCampaignAccess != nil {
		if err := h.AuthorizeCampaignAccess(r, campaignID); err != nil {
			h.writeServiceError(w, err)
			return
		}
	}
	rows, err := h.ConversionMappings.ListCampaignConversionMappings(r.Context(), campaignID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	if rows == nil {
		rows = []ConversionMappingDTO{}
	}
	httpresponse.JSON(w, http.StatusOK, ConversionMappingListResponse{Mappings: rows})
}

func (h *CampaignsHTTPHandlers) putConversionMappings(w http.ResponseWriter, r *http.Request) {
	campaignID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid campaign id")
		return
	}
	if h.AuthorizeCampaignAccess != nil {
		if err := h.AuthorizeCampaignAccess(r, campaignID); err != nil {
			h.writeServiceError(w, err)
			return
		}
	}
	req, ok := coldpath.DecodeRequestOrBadRequest[ReplaceConversionMappingsRequest](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	rows, err := h.ConversionMappings.ReplaceCampaignConversionMappings(r.Context(), campaignID, req.Mappings)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	if rows == nil {
		rows = []ConversionMappingDTO{}
	}
	httpresponse.JSON(w, http.StatusOK, ConversionMappingListResponse{Mappings: rows})
}
