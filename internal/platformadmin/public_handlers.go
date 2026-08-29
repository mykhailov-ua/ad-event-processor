package platformadmin

import (
	"context"
	"net/http"

	ctrlhttp "ad-event-processor/internal/control/http"
	"ad-event-processor/internal/identity"
	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"
)

type PublicActivation interface {
	ActivateOwner(ctx context.Context, req ActivateOwnerRequest) (ActivatedOwner, error)
	AcceptTeamInvite(ctx context.Context, req AcceptTeamInviteRequest) (ActivatedOwner, error)
}

type PublicHTTPHandlers struct {
	Activation        PublicActivation
	AuthClient        ctrlhttp.AuthClientAPI
	ApplyRateLimit    func(http.HandlerFunc) http.HandlerFunc
	WriteServiceError func(http.ResponseWriter, error)
	PolicyRefresh     ctrlhttp.LoginPolicyRefresher
}

type activateRequest struct {
	LicenseToken string `json:"license_token"`
	Email        string `json:"email"`
	Password     string `json:"password"`
	TeamName     string `json:"team_name"`
}

type acceptInviteRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

func (h *PublicHTTPHandlers) Register(mux *http.ServeMux) {
	if h == nil || h.Activation == nil {
		return
	}
	limit := h.ApplyRateLimit
	if limit == nil {
		limit = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	mux.HandleFunc("POST /api/v1/public/activate", limit(h.postActivate))
	mux.HandleFunc("POST /api/v1/public/invite/accept", limit(h.postAcceptInvite))
}

func (h *PublicHTTPHandlers) postActivate(w http.ResponseWriter, r *http.Request) {
	req, ok := coldpath.DecodeRequestOrBadRequest[activateRequest](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	if req.LicenseToken == "" || req.Email == "" || req.Password == "" || req.TeamName == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid activate request")
		return
	}
	owner, err := h.Activation.ActivateOwner(r.Context(), ActivateOwnerRequest{
		LicenseToken: req.LicenseToken,
		Email:        req.Email,
		Password:     req.Password,
		TeamName:     req.TeamName,
	})
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	h.finishLogin(w, r, owner.Email, req.Password)
}

func (h *PublicHTTPHandlers) postAcceptInvite(w http.ResponseWriter, r *http.Request) {
	req, ok := coldpath.DecodeRequestOrBadRequest[acceptInviteRequest](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	if req.Token == "" || req.Password == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid invite accept request")
		return
	}
	owner, err := h.Activation.AcceptTeamInvite(r.Context(), AcceptTeamInviteRequest{
		Token:    req.Token,
		Password: req.Password,
	})
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	h.finishLogin(w, r, owner.Email, req.Password)
}

func (h *PublicHTTPHandlers) finishLogin(w http.ResponseWriter, r *http.Request, email, password string) {
	if h.AuthClient == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "auth not configured")
		return
	}
	resp, err := h.AuthClient.Login(identity.WithHTTPRequest(r.Context(), r), email, password, 1)
	if err != nil {
		httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "login failed after activation")
		return
	}
	if err := ctrlhttp.WriteSessionCookies(w, r, resp.AccessToken, resp.RefreshToken); err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "internal system failure")
		return
	}
	role := ctrlhttp.NormalizeRole(resp.User.Role)
	userDTO := ctrlhttp.UserDTO{
		ID:          resp.User.ID.String(),
		Email:       resp.User.Email,
		Role:        role,
		CustomerID:  resp.User.CustomerID.String(),
		Permissions: ctrlhttp.GetPermissionsForRole(resp.User.Role),
	}
	if h.PolicyRefresh != nil {
		h.PolicyRefresh.RefreshUserPolicy(resp.User.ID, role)
	}
	httpresponse.JSON(w, http.StatusOK, ctrlhttp.LoginResponse{User: userDTO})
}

func (h *PublicHTTPHandlers) writeServiceError(w http.ResponseWriter, err error) {
	if h.WriteServiceError != nil {
		h.WriteServiceError(w, err)
		return
	}
	httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL", "internal error")
}

var _ PublicActivation = (*publicActivationAdapter)(nil)

type publicActivationAdapter struct {
	host activationBridgeHost
}

type activationBridgeHost interface {
	ActivationHost
	InviteAcceptHost
}

func NewPublicActivation(host activationBridgeHost) PublicActivation {
	if host == nil {
		return nil
	}
	return &publicActivationAdapter{host: host}
}

func (a *publicActivationAdapter) ActivateOwner(ctx context.Context, req ActivateOwnerRequest) (ActivatedOwner, error) {
	return ActivateOwner(ctx, a.host, req)
}

func (a *publicActivationAdapter) AcceptTeamInvite(ctx context.Context, req AcceptTeamInviteRequest) (ActivatedOwner, error) {
	return AcceptTeamInvite(ctx, a.host, req)
}
