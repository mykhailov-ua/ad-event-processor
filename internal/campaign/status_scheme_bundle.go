package campaign

import (
	"net/http"

	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
)

func (h *CampaignsHTTPHandlers) registerStatusSchemeRoutes(mux *http.ServeMux, limit func(http.HandlerFunc) http.HandlerFunc, perm func([]string, http.HandlerFunc) http.HandlerFunc) {
	if h == nil || h.StatusSchemes == nil {
		return
	}
	read := []string{"campaigns:read", "campaigns:read:masked"}
	mux.HandleFunc("GET /api/v1/campaigns/{id}/status-schemes", limit(perm(read, h.getStatusSchemes)))
	mux.HandleFunc("POST /api/v1/campaigns/{id}/status-schemes", limit(perm([]string{"campaigns:write"}, h.postStatusSchemes)))
	mux.HandleFunc("PATCH /api/v1/campaigns/{id}/status-schemes/{rule_id}", limit(perm([]string{"campaigns:write"}, h.patchStatusSchemeRule)))
}

func (h *CampaignsHTTPHandlers) getStatusSchemes(w http.ResponseWriter, r *http.Request) {
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
	rows, err := h.StatusSchemes.ListCampaignStatusSchemeRules(r.Context(), campaignID)
	if err != nil {
		h.WriteHandlerError(w, err)
		return
	}
	if rows == nil {
		rows = []StatusSchemeRuleDTO{}
	}
	httpresponse.JSON(w, http.StatusOK, StatusSchemeListResponse{Rules: rows})
}

func (h *CampaignsHTTPHandlers) postStatusSchemes(w http.ResponseWriter, r *http.Request) {
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
	req, ok := coldpath.DecodeRequestOrBadRequest[ReplaceStatusSchemeRequest](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	rows, err := h.StatusSchemes.ReplaceCampaignStatusSchemeRules(r.Context(), campaignID, req.Rules)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	if rows == nil {
		rows = []StatusSchemeRuleDTO{}
	}
	httpresponse.JSON(w, http.StatusOK, StatusSchemeListResponse{Rules: rows})
}

func (h *CampaignsHTTPHandlers) patchStatusSchemeRule(w http.ResponseWriter, r *http.Request) {
	campaignID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid campaign id")
		return
	}
	ruleID, err := uuid.Parse(r.PathValue("rule_id"))
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid rule id")
		return
	}
	if h.AuthorizeCampaignAccess != nil {
		if err := h.AuthorizeCampaignAccess(r, campaignID); err != nil {
			h.WriteHandlerError(w, err)
			return
		}
	}
	req, ok := coldpath.DecodeRequestOrBadRequest[PatchStatusSchemeRuleRequest](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	row, err := h.StatusSchemes.PatchCampaignStatusSchemeRule(r.Context(), campaignID, ruleID, req)
	if err != nil {
		h.WriteHandlerError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, row)
}
