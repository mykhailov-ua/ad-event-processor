package management

import (
	"net/http"

	"espx/pkg/coldpath"
	"espx/pkg/httpresponse"
)

func (h *Handler) postCampaignConsentRequirements(w http.ResponseWriter, r *http.Request) {
	campaignID, err := coldpath.ParsePathUUID(r, "id")
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid campaign id")
		return
	}
	req, ok := coldpath.DecodeRequestOrBadRequest[struct {
		RequireConsentPurposes int16 `json:"require_consent_purposes"`
	}](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	if err := h.svc.UpdateCampaignConsentRequirements(r.Context(), campaignID, req.RequireConsentPurposes); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) postPrivacyErasure(w http.ResponseWriter, r *http.Request) {
	req, ok := coldpath.DecodeRequestOrBadRequest[struct {
		UserID string `json:"user_id"`
	}](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	if req.UserID == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "user_id required")
		return
	}
	id, err := h.svc.CreatePrivacyErasureRequest(r.Context(), req.UserID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusAccepted, map[string]string{"request_id": id.String()})
}
