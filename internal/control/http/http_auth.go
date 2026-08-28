package http

import (
	"context"
	"crypto/subtle"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/identity"
	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

var adminAPIKeyNamespace = uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")

func APIKeyPrincipalID(apiKey string) uuid.UUID {
	return uuid.NewSHA1(adminAPIKeyNamespace, []byte(apiKey))
}

type AuthClientAPI interface {
	Login(ctx context.Context, email, password string, durationHours int32) (identity.LoginResult, error)
	Register(ctx context.Context, adminAPIKey, email, password, role, customerID string) (identity.RegisterResult, error)
	RefreshToken(ctx context.Context, refreshToken string) (identity.RefreshResult, error)
	RevokeToken(ctx context.Context, refreshToken string) error
}

type RegisterGate interface {
	RequirePermission(permission string) func(http.HandlerFunc) http.HandlerFunc
}

type LoginPolicyRefresher interface {
	RefreshUserPolicy(userID uuid.UUID, role string)
}

type RedisShardPicker func([]redis.UniversalClient) redis.UniversalClient

type AuthHandler struct {
	authClient     AuthClientAPI
	tokenMaker     identity.Maker
	redisShards    []redis.UniversalClient
	cfg            *config.Config
	registerGate   RegisterGate
	policyRefresh  LoginPolicyRefresher
	pickRedisShard RedisShardPicker
}

func NewAuthHandler(
	authClient AuthClientAPI,
	tokenMaker identity.Maker,
	redisShards []redis.UniversalClient,
	cfg *config.Config,
	registerGate RegisterGate,
	policyRefresh LoginPolicyRefresher,
	pickRedisShard RedisShardPicker,
) *AuthHandler {
	return &AuthHandler{
		authClient:     authClient,
		tokenMaker:     tokenMaker,
		redisShards:    redisShards,
		cfg:            cfg,
		registerGate:   registerGate,
		policyRefresh:  policyRefresh,
		pickRedisShard: pickRedisShard,
	}
}

func (h *AuthHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/auth/login", h.login)
	mux.HandleFunc("POST /api/v1/auth/logout", h.logout)
	mux.HandleFunc("POST /api/v1/auth/refresh", h.refresh)
	mux.HandleFunc("GET /api/v1/auth/me", h.me)
	if h.registerGate != nil {
		mux.HandleFunc("POST /api/v1/auth/register", h.registerGate.RequirePermission(PermUsersWrite)(h.register))
	} else {
		mux.HandleFunc("POST /api/v1/auth/register", h.requireAdminKeyFallback(h.register))
	}
}

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

func requestIsSecure(r *http.Request) bool {
	if r == nil {
		return false
	}
	return r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

func setCookie(w http.ResponseWriter, r *http.Request, name, value, path string, maxAge int, httpOnly bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     path,
		MaxAge:   maxAge,
		HttpOnly: httpOnly,
		Secure:   requestIsSecure(r),
		SameSite: http.SameSiteStrictMode,
	})
}

func issueCSRFToken(w http.ResponseWriter, r *http.Request) error {
	if cookie, err := r.Cookie("csrfToken"); err == nil && cookie.Value != "" {
		w.Header().Set("X-CSRF-Token", cookie.Value)
		return nil
	}
	csrf, err := GenerateSecureToken(32)
	if err != nil {
		return err
	}
	setCookie(w, r, "csrfToken", csrf, "/", 3600, false)
	w.Header().Set("X-CSRF-Token", csrf)
	return nil
}

func rotateCSRFToken(w http.ResponseWriter, r *http.Request) error {
	csrf, err := GenerateSecureToken(32)
	if err != nil {
		return err
	}
	setCookie(w, r, "csrfToken", csrf, "/", 3600, false)
	w.Header().Set("X-CSRF-Token", csrf)
	return nil
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

	setCookie(w, r, "accessToken", resp.AccessToken, "/", 3600, true)
	setCookie(w, r, "refreshToken", resp.RefreshToken, "/api/v1/auth", 30*24*3600, true)
	if err := rotateCSRFToken(w, r); err != nil {
		slog.Error("failed to generate secure csrf token due to entropy starvation", "error", err)
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "internal system failure")
		return
	}

	userDTO := UserDTO{
		ID:          resp.User.ID.String(),
		Email:       resp.User.Email,
		Role:        NormalizeRole(resp.User.Role),
		CustomerID:  resp.User.CustomerID.String(),
		Permissions: GetPermissionsForRole(resp.User.Role),
	}
	if h.policyRefresh != nil {
		h.policyRefresh.RefreshUserPolicy(resp.User.ID, userDTO.Role)
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
			if errRev := identity.RevokeTokenSessionShards(r.Context(), h.redisShards, payload.ID, payload.SessionID, ttl); errRev != nil {
				slog.Error("failed to revoke tokens on logout", "error", errRev)
			}
		}
	}

	setCookie(w, r, "accessToken", "", "/", -1, true)
	setCookie(w, r, "refreshToken", "", "/api/v1/auth", -1, true)
	setCookie(w, r, "csrfToken", "", "/", -1, false)
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

	setCookie(w, r, "accessToken", resp.AccessToken, "/", 3600, true)
	setCookie(w, r, "refreshToken", resp.RefreshToken, "/api/v1/auth", 30*24*3600, true)
	if err := rotateCSRFToken(w, r); err != nil {
		slog.Error("failed to rotate csrf token on refresh", "error", err)
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "internal system failure")
		return
	}

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

	if len(h.redisShards) > 0 && h.pickRedisShard != nil {
		redisClient := h.pickRedisShard(h.redisShards)
		if redisClient != nil {
			revoked, errRev := identity.CheckTokenRevocation(r.Context(), redisClient, payload)
			if errRev != nil {
				if h.cfg != nil && h.cfg.Env == "development" {
					slog.Warn("redis revocation check failed in development, allowing /me", "error", errRev)
				} else {
					slog.Error("redis revocation check failed on /me, blocking request", "error", errRev)
					httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized: security check failed")
					return
				}
			} else if revoked {
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

	if err := issueCSRFToken(w, r); err != nil {
		slog.Error("failed to issue csrf token on /me", "error", err)
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "internal system failure")
		return
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
