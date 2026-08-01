package controlplane

import (
	"crypto/subtle"
	"log/slog"
	"net/http"
	"time"

	"espx/internal/config"
	"espx/internal/identity"
	"espx/pkg/coldpath"
	"espx/pkg/httpresponse"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

var adminAPIKeyNamespace = uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")

func apiKeyPrincipalID(apiKey string) uuid.UUID {
	return uuid.NewSHA1(adminAPIKeyNamespace, []byte(apiKey))
}

type AuthHandler struct {
	authClient     *AuthClient
	tokenMaker     identity.Maker
	rdbs           []redis.UniversalClient
	cfg            *config.Config
	authMiddleware *AuthMiddleware
}

func NewAuthHandler(authClient *AuthClient, tokenMaker identity.Maker, rdbs []redis.UniversalClient, cfg *config.Config, authMiddleware *AuthMiddleware) *AuthHandler {
	return &AuthHandler{
		authClient:     authClient,
		tokenMaker:     tokenMaker,
		rdbs:           rdbs,
		cfg:            cfg,
		authMiddleware: authMiddleware,
	}
}

func (h *AuthHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/auth/login", h.login)
	mux.HandleFunc("POST /api/v1/auth/logout", h.logout)
	mux.HandleFunc("POST /api/v1/auth/refresh", h.refresh)
	mux.HandleFunc("GET /api/v1/auth/me", h.me)
	if h.authMiddleware != nil {
		mux.HandleFunc("POST /api/v1/auth/register", h.authMiddleware.RequirePermission(PermUsersWrite)(h.register))
	} else {
		// FIX [1.6]: authMiddleware nil means the service is not fully wired
		// (e.g. installer mode). Fall back to requiring the static admin API key
		// so the endpoint is never truly open.
		mux.HandleFunc("POST /api/v1/auth/register", h.requireAdminKeyFallback(h.register))
	}
}

// requireAdminKeyFallback guards a handler with a constant-time admin-key check
// when the full auth middleware is unavailable.
func (h *AuthHandler) requireAdminKeyFallback(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if h.cfg == nil || len(h.cfg.AdminAPIKey) == 0 {
			httpresponse.Error(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "auth not configured")
			return
		}
		key := r.Header.Get("X-Admin-API-Key")
		if subtle.ConstantTimeCompare([]byte(key), []byte(h.cfg.AdminAPIKey)) != 1 {
			httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
			return
		}
		next(w, r)
	}
}

func setCookie(w http.ResponseWriter, name, value, path string, maxAge int, httpOnly bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     path,
		MaxAge:   maxAge,
		HttpOnly: httpOnly,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UserDTO struct {
	ID          string   `json:"id"`
	Email       string   `json:"email,omitempty"`
	Role        string   `json:"role"`
	CustomerID  string   `json:"customer_id"`
	Permissions []string `json:"permissions,omitempty"`
}

type LoginResponse struct {
	User UserDTO `json:"user"`
}

type RefreshResponse struct {
	Status string `json:"status"`
}

type RegisterResponse struct {
	UserID string `json:"user_id"`
}

func (h *AuthHandler) login(w http.ResponseWriter, r *http.Request) {
	// FIX [1.9]: cap body at 64 KB to prevent memory exhaustion.
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "failed to read request body")
		return
	}

	req, err := coldpath.DecodeBody[LoginRequest](body)
	if err != nil || req.Email == "" || req.Password == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid login request")
		return
	}

	resp, err := h.authClient.Login(identity.WithHTTPRequest(r.Context(), r), req.Email, req.Password, 1)
	if err != nil {
		slog.Warn("login failed", "email", req.Email, "error", err)
		httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid credentials")
		return
	}

	setCookie(w, "accessToken", resp.AccessToken, "/", 3600, true)
	setCookie(w, "refreshToken", resp.RefreshToken, "/api/v1/auth", 30*24*3600, true)
	csrf, err := GenerateSecureToken(32)
	if err != nil {
		slog.Error("failed to generate secure csrf token due to entropy starvation", "error", err)
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "internal system failure")
		return
	}
	setCookie(w, "csrfToken", csrf, "/", 3600, false)
	w.Header().Set("X-CSRF-Token", csrf)

	userDTO := UserDTO{
		ID:          resp.User.ID.String(),
		Email:       resp.User.Email,
		Role:        NormalizeRole(resp.User.Role),
		CustomerID:  resp.User.CustomerID.String(),
		Permissions: GetPermissionsForRole(resp.User.Role),
	}
	if h.authMiddleware != nil && h.authMiddleware.policy != nil {
		h.authMiddleware.policy.RefreshUser(resp.User.ID, userDTO.Role)
	}

	httpresponse.JSON(w, http.StatusOK, LoginResponse{User: userDTO})
}

func (h *AuthHandler) logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refreshToken")
	if err == nil && cookie.Value != "" {
		if errRevoke := h.authClient.RevokeToken(r.Context(), cookie.Value); errRevoke != nil {
			slog.Warn("failed to revoke token on logout", "error", errRevoke)
		}
	}

	accessCookie, err := r.Cookie("accessToken")
	if err == nil && accessCookie.Value != "" {
		payload, errPayload := h.tokenMaker.VerifyToken(accessCookie.Value)
		if errPayload == nil {
			ttl := time.Until(payload.ExpiredAt)
			if errRev := identity.RevokeTokenSessionShards(r.Context(), h.rdbs, payload.ID, payload.SessionID, ttl); errRev != nil {
				slog.Error("failed to revoke tokens on logout", "error", errRev)
			}
		}
	}

	setCookie(w, "accessToken", "", "/", -1, true)
	setCookie(w, "refreshToken", "", "/api/v1/auth", -1, true)
	setCookie(w, "csrfToken", "", "/", -1, false)
	httpresponse.JSON(w, http.StatusNoContent, nil)
}

func (h *AuthHandler) refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refreshToken")
	if err != nil || cookie.Value == "" {
		httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing refresh token")
		return
	}

	resp, err := h.authClient.RefreshToken(r.Context(), cookie.Value)
	if err != nil {
		slog.Warn("refresh token failed", "error", err)
		httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}

	setCookie(w, "accessToken", resp.AccessToken, "/", 3600, true)
	setCookie(w, "refreshToken", resp.RefreshToken, "/api/v1/auth", 30*24*3600, true)

	httpresponse.JSON(w, http.StatusOK, RefreshResponse{Status: "refreshed"})
}

func (h *AuthHandler) me(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("accessToken")
	if err != nil || cookie.Value == "" {
		httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}

	payload, err := h.tokenMaker.VerifyToken(cookie.Value)
	if err != nil {
		httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}

	if len(h.rdbs) > 0 {
		rdb := PickHealthyControlShard(h.rdbs)
		if rdb != nil {
			revoked, errRev := identity.CheckTokenRevocation(r.Context(), rdb, payload)
			if errRev != nil {
				slog.Error("redis revocation check failed on /me, blocking request", "error", errRev)
				httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized: security check failed")
				return
			}
			if revoked {
				httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized: token revoked")
				return
			}
		}
	}

	dto := UserDTO{
		ID:          payload.UserID.String(),
		Role:        NormalizeRole(payload.Role),
		CustomerID:  payload.CustomerID.String(),
		Permissions: GetPermissionsForRole(payload.Role),
	}

	httpresponse.JSON(w, http.StatusOK, dto)
}

type RegisterRequest struct {
	Email      string `json:"email"`
	Password   string `json:"password"`
	Role       string `json:"role"`
	CustomerID string `json:"customer_id,omitempty"`
}

func (h *AuthHandler) register(w http.ResponseWriter, r *http.Request) {
	// FIX [1.9]: cap body at 64 KB.
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "failed to read request body")
		return
	}

	req, err := coldpath.DecodeBody[RegisterRequest](body)
	if err != nil || req.Email == "" || req.Password == "" || req.Role == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid register request")
		return
	}

	reqRole := NormalizeRole(req.Role)
	if reqRole != RoleManager && reqRole != RoleUser {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "role must be M or U")
		return
	}
	if reqRole == RoleUser && req.CustomerID == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "customer_id is required for user role")
		return
	}

	resp, err := h.authClient.Register(r.Context(), string(h.cfg.AdminAPIKey), req.Email, req.Password, reqRole, req.CustomerID)
	if err != nil {
		slog.Warn("registration failed", "email", req.Email, "error", err)
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "registration failed")
		return
	}

	httpresponse.JSON(w, http.StatusCreated, RegisterResponse{UserID: resp.UserID.String()})
}
