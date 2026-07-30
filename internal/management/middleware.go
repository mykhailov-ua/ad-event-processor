package management

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"net/http"
	"strings"

	"espx/internal/auth"
	"espx/internal/config"
	"espx/pkg/httpresponse"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type contextKey string

const UserContextKey contextKey = "authenticated_user"

type AuthenticatedUser struct {
	UserID     uuid.UUID
	Role       string
	CustomerID uuid.UUID
	AuthSource string
}

func (u AuthenticatedUser) IsUser() bool {
	return u.Role == RoleUser
}

func GetUser(ctx context.Context) (AuthenticatedUser, bool) {
	u, ok := ctx.Value(UserContextKey).(AuthenticatedUser)
	return u, ok
}

type AuthMiddleware struct {
	tokenMaker    auth.Maker
	rdb           redis.UniversalClient
	controlRdbs   []redis.UniversalClient
	cfg           *config.Config
	authClient    *AuthClient
	apiKeyLimiter *apiKeyRateLimiter
}

func NewAuthMiddleware(tokenMaker auth.Maker, rdb redis.UniversalClient, cfg *config.Config, authClient *AuthClient) *AuthMiddleware {
	return NewAuthMiddlewareShards(tokenMaker, rdb, nil, cfg, authClient)
}

func NewAuthMiddlewareShards(tokenMaker auth.Maker, rdb redis.UniversalClient, controlRdbs []redis.UniversalClient, cfg *config.Config, authClient *AuthClient) *AuthMiddleware {
	rps := defaultAPIKeyRPS
	burst := defaultAPIKeyBurst
	if cfg != nil && cfg.SelfServeAPIKeyRPS > 0 {
		rps = cfg.SelfServeAPIKeyRPS
		burst = int(rps * 2)
		if burst < 1 {
			burst = defaultAPIKeyBurst
		}
	}
	return &AuthMiddleware{
		tokenMaker:    tokenMaker,
		rdb:           rdb,
		controlRdbs:   controlRdbs,
		cfg:           cfg,
		authClient:    authClient,
		apiKeyLimiter: newAPIKeyRateLimiter(rps, burst),
	}
}

func (m *AuthMiddleware) RequirePermission(permission string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			user, ok := m.authenticate(w, r)
			if !ok {
				return
			}
			if !HasPermission(user.Role, permission) {
				httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "forbidden: insufficient permissions")
				return
			}
			ctx := context.WithValue(r.Context(), UserContextKey, user)
			next(w, r.WithContext(ctx))
		}
	}
}

func (m *AuthMiddleware) RequireSelfServe(permission string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if key := strings.TrimSpace(r.Header.Get("X-API-Key")); key != "" {
				user, ok := m.authenticateAPIKey(w, r, key)
				if !ok {
					return
				}
				if !HasPermission(user.Role, permission) {
					httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "forbidden: insufficient permissions")
					return
				}
				ctx := context.WithValue(r.Context(), UserContextKey, user)
				next(w, r.WithContext(ctx))
				return
			}
			user, ok := m.authenticate(w, r)
			if !ok {
				return
			}
			if !HasPermission(user.Role, permission) {
				httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "forbidden: insufficient permissions")
				return
			}
			ctx := context.WithValue(r.Context(), UserContextKey, user)
			next(w, r.WithContext(ctx))
		}
	}
}

func (m *AuthMiddleware) RequireAuth(allowedRoles ...string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			user, ok := m.authenticate(w, r)
			if !ok {
				return
			}
			roleAllowed := false
			for _, allowed := range allowedRoles {
				if user.Role == NormalizeRole(allowed) || user.Role == RoleAdmin {
					roleAllowed = true
					break
				}
			}
			if !roleAllowed {
				httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "forbidden: insufficient permissions")
				return
			}
			ctx := context.WithValue(r.Context(), UserContextKey, user)
			next(w, r.WithContext(ctx))
		}
	}
}

func apiKeyDigest(rawKey string) string {
	sum := sha256.Sum256([]byte(rawKey))
	return hex.EncodeToString(sum[:])
}

func (m *AuthMiddleware) authenticateAPIKey(w http.ResponseWriter, r *http.Request, rawKey string) (AuthenticatedUser, bool) {
	if m.authClient == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "AUTH_UNAVAILABLE", "auth service not configured")
		return AuthenticatedUser{}, false
	}
	if m.apiKeyLimiter != nil && !m.apiKeyLimiter.allow(apiKeyDigest(rawKey)) {
		httpresponse.Error(w, http.StatusTooManyRequests, "TOO_MANY_REQUESTS", "api key rate limit exceeded")
		return AuthenticatedUser{}, false
	}

	resp, err := m.authClient.VerifyAPIKey(r.Context(), rawKey)
	if err != nil {
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.Unauthenticated:
				httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized: invalid api key")
				return AuthenticatedUser{}, false
			case codes.ResourceExhausted:
				httpresponse.Error(w, http.StatusTooManyRequests, "TOO_MANY_REQUESTS", st.Message())
				return AuthenticatedUser{}, false
			}
		}
		slog.Error("api key verification failed", "error", err)
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL", "failed to verify api key")
		return AuthenticatedUser{}, false
	}
	if resp.User == nil {
		httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized: invalid api key")
		return AuthenticatedUser{}, false
	}

	userID, err := uuid.Parse(resp.User.Id)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL", "invalid user id from auth")
		return AuthenticatedUser{}, false
	}
	customerID := uuid.Nil
	if resp.User.CustomerId != "" {
		customerID, err = uuid.Parse(resp.User.CustomerId)
		if err != nil {
			httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL", "invalid customer id from auth")
			return AuthenticatedUser{}, false
		}
	}

	return AuthenticatedUser{
		UserID:     userID,
		Role:       NormalizeRole(resp.User.Role),
		CustomerID: customerID,
		AuthSource: "api_key",
	}, true
}

func (m *AuthMiddleware) authenticate(w http.ResponseWriter, r *http.Request) (AuthenticatedUser, bool) {
	if key := r.Header.Get("X-Admin-API-Key"); key != "" && m.cfg != nil && key == string(m.cfg.AdminAPIKey) {
		return AuthenticatedUser{
			UserID:     apiKeyPrincipalID(key),
			Role:       RoleAdmin,
			CustomerID: uuid.Nil,
			AuthSource: "api_key",
		}, true
	}

	cookie, err := r.Cookie("accessToken")
	if err != nil || cookie.Value == "" {
		httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized: missing token")
		return AuthenticatedUser{}, false
	}

	payload, err := m.tokenMaker.VerifyToken(cookie.Value)
	if err != nil {
		httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized: invalid token")
		return AuthenticatedUser{}, false
	}

	if len(m.controlRdbs) > 0 || m.rdb != nil {
		rdbs := m.controlRdbs
		if len(rdbs) == 0 {
			rdbs = []redis.UniversalClient{m.rdb}
		}
		revoked, errRev := auth.CheckTokenRevocationShards(r.Context(), rdbs, payload)
		if errRev != nil {
			slog.Error("redis revocation check failed, blocking request to prevent security bypass", "error", errRev)
			httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized: security check failed")
			return AuthenticatedUser{}, false
		}
		if revoked {
			httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized: session revoked")
			return AuthenticatedUser{}, false
		}
	}

	return AuthenticatedUser{
		UserID:     payload.UserID,
		Role:       NormalizeRole(payload.Role),
		CustomerID: payload.CustomerID,
		AuthSource: "session",
	}, true
}
