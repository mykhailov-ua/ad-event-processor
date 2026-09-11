package googlesheets

import (
	"context"
	"net/http"
	"strings"
	"time"

	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
)

type connectionStore interface {
	HasConnection(ctx context.Context, userID uuid.UUID) (bool, error)
	OperatorAccountEmail(ctx context.Context, userID uuid.UUID) (string, error)
	UpsertTokens(ctx context.Context, userID uuid.UUID, refreshToken, accessToken string, expiresAt time.Time, scopes string) error
	DeleteConnection(ctx context.Context, userID uuid.UUID) error
}

type HTTPHandlers struct {
	Config               Config
	Store                connectionStore
	ApplyRateLimit       func(http.HandlerFunc) http.HandlerFunc
	RequirePermission    func(string, http.HandlerFunc) http.HandlerFunc
	RequireAnyPermission func([]string, http.HandlerFunc) http.HandlerFunc
	ActorUserID          func(*http.Request) (uuid.UUID, bool)
	WriteServiceError    func(http.ResponseWriter, error)
	Audit                func(ctx context.Context, actorID uuid.UUID, action string, metadata any)
}

func (h *HTTPHandlers) Register(mux *http.ServeMux) {
	if h == nil || mux == nil {
		return
	}
	limit := h.ApplyRateLimit
	if limit == nil {
		limit = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	perm := h.RequirePermission
	if perm == nil {
		perm = func(_ string, next http.HandlerFunc) http.HandlerFunc { return next }
	}
	permAny := h.RequireAnyPermission
	if permAny == nil {
		permAny = func(perms []string, next http.HandlerFunc) http.HandlerFunc {
			if len(perms) == 0 {
				return next
			}
			return perm(perms[0], next)
		}
	}
	writePerms := []string{"settings:write", "integrations:write"}
	readPerms := []string{"settings:read", "integrations:read"}

	mux.HandleFunc("GET /api/v1/integrations/google-sheets/connect", limit(permAny(writePerms, h.getConnect)))
	mux.HandleFunc("GET /api/v1/integrations/google-sheets/callback", limit(h.getCallback))
	mux.HandleFunc("GET /api/v1/integrations/google-sheets/status", limit(permAny(readPerms, h.getStatus)))
	mux.HandleFunc("DELETE /api/v1/integrations/google-sheets", limit(permAny(writePerms, h.deleteConnection)))
}

func (h *HTTPHandlers) getConnect(w http.ResponseWriter, r *http.Request) {
	if h.Config.ClientID == "" || h.Config.ClientSecret == "" {
		httpresponse.Error(w, http.StatusServiceUnavailable, "NOT_CONFIGURED", "google sheets integration is not configured")
		return
	}
	userID, ok := h.actorID(r)
	if !ok {
		httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}
	state, err := issueOAuthState(h.Config.StateSecret, userID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	url, err := ConnectURL(h.Config, state)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	http.Redirect(w, r, url, http.StatusFound)
}

func (h *HTTPHandlers) getCallback(w http.ResponseWriter, r *http.Request) {
	if h.Config.ClientID == "" || h.Config.ClientSecret == "" {
		httpresponse.Error(w, http.StatusServiceUnavailable, "NOT_CONFIGURED", "google sheets integration is not configured")
		return
	}
	userID, err := verifyOAuthStateReturnUser(h.Config.StateSecret, r.URL.Query().Get("state"))
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid oauth state")
		return
	}
	if sessionUserID, ok := h.actorID(r); ok && sessionUserID != userID {
		httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "oauth state does not match session")
		return
	}
	code := strings.TrimSpace(r.URL.Query().Get("code"))
	if code == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "missing authorization code")
		return
	}
	redirectURI := RedirectURI(h.Config.PublicURL)
	access, refresh, expires, err := exchangeAuthorizationCode(r.Context(), nil, h.Config, code, redirectURI)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	if refresh == "" {
		httpresponse.Error(w, http.StatusBadGateway, "UPSTREAM_ERROR", "google did not return a refresh token")
		return
	}
	if err := h.Store.UpsertTokens(r.Context(), userID, refresh, access, expires, OAuthScope); err != nil {
		h.writeServiceError(w, err)
		return
	}
	if h.Audit != nil {
		h.Audit(r.Context(), userID, "google_sheets.connect", map[string]string{"status": "connected"})
	}
	http.Redirect(w, r, "/integrations/google-sheets?google_sheets=connected", http.StatusFound)
}

func (h *HTTPHandlers) getStatus(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.actorID(r)
	if !ok {
		httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}
	if h.Config.ClientID == "" || h.Config.ClientSecret == "" {
		httpresponse.JSON(w, http.StatusOK, StatusDTO{Connected: false, Message: "not configured"})
		return
	}
	if h.Store == nil {
		httpresponse.JSON(w, http.StatusOK, StatusDTO{Connected: false, Message: "not configured"})
		return
	}
	connected, err := h.Store.HasConnection(r.Context(), userID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	dto := StatusDTO{Connected: connected}
	if connected {
		email, err := h.Store.OperatorAccountEmail(r.Context(), userID)
		if err != nil {
			h.writeServiceError(w, err)
			return
		}
		dto.AccountEmail = email
	}
	httpresponse.JSON(w, http.StatusOK, dto)
}

func (h *HTTPHandlers) deleteConnection(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.actorID(r)
	if !ok {
		httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}
	if h.Store == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "NOT_CONFIGURED", "google sheets integration is not configured")
		return
	}
	if err := h.Store.DeleteConnection(r.Context(), userID); err != nil {
		if err == ErrNotConnected {
			httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", "google sheets is not connected")
			return
		}
		h.writeServiceError(w, err)
		return
	}
	if h.Audit != nil {
		h.Audit(r.Context(), userID, "google_sheets.disconnect", map[string]string{"status": "disconnected"})
	}
	httpresponse.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *HTTPHandlers) actorID(r *http.Request) (uuid.UUID, bool) {
	if h.ActorUserID != nil {
		return h.ActorUserID(r)
	}
	return uuid.Nil, false
}

func (h *HTTPHandlers) writeServiceError(w http.ResponseWriter, err error) {
	if h.WriteServiceError != nil {
		h.WriteServiceError(w, err)
		return
	}
	httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
}
