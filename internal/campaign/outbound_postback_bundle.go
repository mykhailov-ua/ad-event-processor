package campaign

import (
	"net/http"

	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
)

func (h *CampaignsHTTPHandlers) registerOutboundPostbackRoutes(mux *http.ServeMux, limit func(http.HandlerFunc) http.HandlerFunc, perm func([]string, http.HandlerFunc) http.HandlerFunc) {
	if h == nil || h.OutboundPostbacks == nil {
		return
	}
	read := []string{"campaigns:read", "campaigns:read:masked"}
	mux.HandleFunc("GET /api/v1/campaigns/{id}/outbound-postbacks", limit(perm(read, h.getOutboundPostbacks)))
	mux.HandleFunc("POST /api/v1/campaigns/{id}/outbound-postbacks", limit(perm([]string{"campaigns:write"}, h.postOutboundPostbacks)))
	mux.HandleFunc("PATCH /api/v1/campaigns/{id}/outbound-postbacks/{postback_id}", limit(perm([]string{"campaigns:write"}, h.patchOutboundPostback)))
	mux.HandleFunc("POST /api/v1/campaigns/{id}/outbound-postbacks/{postback_id}/test", limit(perm([]string{"campaigns:write"}, h.testOutboundPostback)))
}

func (h *CampaignsHTTPHandlers) getOutboundPostbacks(w http.ResponseWriter, r *http.Request) {
	campaignID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid campaign id")
		return
	}
	if h.AuthorizeCampaignAccess != nil {
		if err := h.AuthorizeCampaignAccess(r, campaignID); err != nil {
			h.WriteHandlerError(w, err)
			return
		}
	}
	rows, err := h.OutboundPostbacks.ListCampaignOutboundPostbacks(r.Context(), campaignID)
	if err != nil {
		h.WriteHandlerError(w, err)
		return
	}
	if rows == nil {
		rows = []OutboundPostbackDTO{}
	}
	httpresponse.JSON(w, http.StatusOK, OutboundPostbackListResponse{Postbacks: rows})
}

func (h *CampaignsHTTPHandlers) postOutboundPostbacks(w http.ResponseWriter, r *http.Request) {
	campaignID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid campaign id")
		return
	}
	if h.AuthorizeCampaignAccess != nil {
		if err := h.AuthorizeCampaignAccess(r, campaignID); err != nil {
			h.WriteHandlerError(w, err)
			return
		}
	}
	req, ok := coldpath.DecodeRequestOrBadRequest[ReplaceOutboundPostbacksRequest](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	rows, err := h.OutboundPostbacks.ReplaceCampaignOutboundPostbacks(r.Context(), campaignID, req.Postbacks)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	if rows == nil {
		rows = []OutboundPostbackDTO{}
	}
	httpresponse.JSON(w, http.StatusOK, OutboundPostbackListResponse{Postbacks: rows})
}

func (h *CampaignsHTTPHandlers) patchOutboundPostback(w http.ResponseWriter, r *http.Request) {
	campaignID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid campaign id")
		return
	}
	postbackID, err := uuid.Parse(r.PathValue("postback_id"))
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid postback id")
		return
	}
	if h.AuthorizeCampaignAccess != nil {
		if err := h.AuthorizeCampaignAccess(r, campaignID); err != nil {
			h.WriteHandlerError(w, err)
			return
		}
	}
	req, ok := coldpath.DecodeRequestOrBadRequest[PatchOutboundPostbackRequest](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	row, err := h.OutboundPostbacks.PatchCampaignOutboundPostback(r.Context(), campaignID, postbackID, req)
	if err != nil {
		h.WriteHandlerError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, row)
}

func (h *CampaignsHTTPHandlers) testOutboundPostback(w http.ResponseWriter, r *http.Request) {
	campaignID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid campaign id")
		return
	}
	postbackID, err := uuid.Parse(r.PathValue("postback_id"))
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid postback id")
		return
	}
	if h.AuthorizeCampaignAccess != nil {
		if err := h.AuthorizeCampaignAccess(r, campaignID); err != nil {
			h.WriteHandlerError(w, err)
			return
		}
	}
	result, err := h.OutboundPostbacks.DryRunCampaignOutboundPostback(r.Context(), campaignID, postbackID)
	if err != nil {
		h.WriteHandlerError(w, err)
		return
	}
	status := http.StatusOK
	if !result.OK {
		status = http.StatusUnprocessableEntity
	}
	httpresponse.JSON(w, status, result)
}
