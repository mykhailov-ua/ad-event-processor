package campaign

import (
	"net/http"
	"strings"

	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
)

func (h *CampaignsHTTPHandlers) registerTrackerClickRoutes(mux *http.ServeMux, limit func(http.HandlerFunc) http.HandlerFunc, perm func([]string, http.HandlerFunc) http.HandlerFunc) {
	h.RegisterTrackerClickRoutes(mux, limit, perm)
}

// RegisterTrackerClickRoutes wires POST /api/v1/tracker/clicks (programmatic click mint).
func (h *CampaignsHTTPHandlers) RegisterTrackerClickRoutes(mux *http.ServeMux, limit func(http.HandlerFunc) http.HandlerFunc, perm func([]string, http.HandlerFunc) http.HandlerFunc) {
	if h == nil {
		return
	}
	if limit == nil {
		limit = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	if perm == nil {
		perm = func(_ []string, next http.HandlerFunc) http.HandlerFunc { return next }
	}
	read := []string{"campaigns:read", "campaigns:read:masked", "campaigns:write"}
	mux.HandleFunc("POST /api/v1/tracker/clicks", limit(perm(read, h.postMintProgrammaticClick)))
}

func (h *CampaignsHTTPHandlers) postMintProgrammaticClick(w http.ResponseWriter, r *http.Request) {
	req, ok := coldpath.DecodeRequestOrBadRequest[ProgrammaticClickMintRequest](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	campaignID, err := ValidateProgrammaticClickMintRequest(req)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	if h.AuthorizeCampaignAccess != nil {
		if err := h.AuthorizeCampaignAccess(r, campaignID); err != nil {
			if h.WriteServiceError != nil {
				h.WriteServiceError(w, err)
				return
			}
			httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "forbidden")
			return
		}
	}
	base := ""
	if h.TrackerPublicBaseURL != nil {
		base = h.TrackerPublicBaseURL()
	}
	signing := ProgrammaticClickLinkSigning{}
	if h.Campaigns != nil {
		camp, campErr := h.Campaigns.GetCampaign(r.Context(), campaignID)
		if campErr == nil && camp.LinkSigningEnabled {
			signing.Enabled = true
			signing.TTLSec = camp.LinkSigningTTLSec
			signing.AttestationMode = camp.AttestationMode
			signing.AttestationEnabled = camp.AttestationEnabled
		}
	}
	if h.LinkSigningSecret != nil {
		signing.Secret = h.LinkSigningSecret()
	}
	resp, err := MintProgrammaticClick(base, campaignID, req, signing)
	if err != nil {
		if IsValidationError(err) {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			return
		}
		if h.WriteServiceError != nil {
			h.WriteServiceError(w, err)
			return
		}
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL", err.Error())
		return
	}
	if h.CampaignAuditLog != nil {
		adminID := uuid.Nil
		if user, ok := authz.GetUser(r.Context()); ok {
			adminID = user.UserID
		}
		target := campaignID
		h.CampaignAuditLog(r.Context(), adminID, "MINT_PROGRAMMATIC_CLICK", "campaign", &target, map[string]string{
			"click_id": resp.ClickID,
		}, map[string]string{
			"auth_source": authSourceFromRequest(r),
		})
	}
	httpresponse.JSON(w, http.StatusOK, resp)
}

func authSourceFromRequest(r *http.Request) string {
	if strings.TrimSpace(r.Header.Get("X-API-Key")) != "" {
		return "api_key"
	}
	if user, ok := authz.GetUser(r.Context()); ok && user.AuthSource != "" {
		return user.AuthSource
	}
	return "session"
}
