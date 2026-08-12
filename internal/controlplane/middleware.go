package controlplane

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/controlplane/authz"
	"github.com/bidshard/ad-event-processor/internal/identity"
	"github.com/bidshard/ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
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
	Scope      authz.Scope
}

func (u AuthenticatedUser) IsUser() bool {
	return u.Role == RoleUser
}

func (u AuthenticatedUser) IsBuyer() bool {
	return u.Role == RoleBuyer
}

func (u AuthenticatedUser) HasBoundCustomer() bool {
	return u.IsUser() || u.IsBuyer()
}

func GetUser(ctx context.Context) (AuthenticatedUser, bool) {
	u, ok := ctx.Value(UserContextKey).(AuthenticatedUser)
	return u, ok
}

type AuthMiddleware struct {
	tokenMaker    identity.Maker
	rdb           redis.UniversalClient
	controlRdbs   []redis.UniversalClient
	cfg           *config.Config
	authClient    *AuthClient
	apiKeyLimiter *apiKeyRateLimiter
	policy        *authz.Store
	pool          *pgxpool.Pool
}

func NewAuthMiddleware(tokenMaker identity.Maker, rdb redis.UniversalClient, cfg *config.Config, authClient *AuthClient) *AuthMiddleware {
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
		cfg:           cfg,
		authClient:    authClient,
		apiKeyLimiter: newAPIKeyRateLimiter(rps, burst),
	}
}

func (m *AuthMiddleware) SetControlRedisShards(rdbs []redis.UniversalClient) {
	m.controlRdbs = rdbs
}

func (m *AuthMiddleware) controlRedis() []redis.UniversalClient {
	if len(m.controlRdbs) > 0 {
		return m.controlRdbs
	}
	if m.rdb != nil {
		return []redis.UniversalClient{m.rdb}
	}
	return nil
}

func (m *AuthMiddleware) SetPolicyStore(store *authz.Store) {
	m.policy = store
}

func (m *AuthMiddleware) SetPool(pool *pgxpool.Pool) {
	m.pool = pool
}

func (m *AuthMiddleware) attachAuthz(ctx context.Context, user AuthenticatedUser) context.Context {
	if m.policy == nil {
		return context.WithValue(ctx, UserContextKey, user)
	}
	snap := m.policy.EffectivePermissionsDB(ctx, m.pool, user.UserID, user.Role)
	if user.Scope == "" {
		user.Scope = snap.Scope
	}
	if user.AuthSource == "api_key" && user.CustomerID != uuid.Nil {
		user.Scope = authz.ScopeCustomer
	}
	ctx = authz.WithSnapshot(ctx, snap)
	return context.WithValue(ctx, UserContextKey, user)
}

func (m *AuthMiddleware) checkPermission(ctx context.Context, user AuthenticatedUser, permission string) bool {
	if snap, ok := authz.SnapshotFromContext(ctx); ok {
		return snap.Has(permission)
	}
	return HasPermission(user.Role, permission)
}

func (m *AuthMiddleware) RequirePermission(permission string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			user, ok := m.authenticate(w, r)
			if !ok {
				return
			}
			ctx := m.attachAuthz(r.Context(), user)
			if !m.checkPermission(ctx, user, permission) {
				httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "forbidden: insufficient permissions")
				return
			}
			next(w, r.WithContext(ctx))
		}
	}
}

func (m *AuthMiddleware) RequireAnyPermission(permissions ...string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			user, ok := m.authenticate(w, r)
			if !ok {
				return
			}
			ctx := m.attachAuthz(r.Context(), user)
			allowed := false
			for _, p := range permissions {
				if m.checkPermission(ctx, user, p) {
					allowed = true
					break
				}
			}
			if !allowed {
				httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "forbidden: insufficient permissions")
				return
			}
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
				user.Scope = authz.ScopeCustomer
				ctx := m.attachAuthz(r.Context(), user)
				if !m.checkPermission(ctx, user, permission) {
					httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "forbidden: insufficient permissions")
					return
				}
				next(w, r.WithContext(ctx))
				return
			}
			user, ok := m.authenticate(w, r)
			if !ok {
				return
			}
			ctx := m.attachAuthz(r.Context(), user)
			if !m.checkPermission(ctx, user, permission) {
				httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "forbidden: insufficient permissions")
				return
			}
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
			ctx := m.attachAuthz(r.Context(), user)
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

	user, err := m.authClient.VerifyAPIKey(r.Context(), rawKey)
	if err != nil {
		if errors.Is(err, identity.ErrInvalidAPIKey) || errors.Is(err, identity.ErrInvalidCredentials) {
			httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized: invalid api key")
			return AuthenticatedUser{}, false
		}
		if errors.Is(err, identity.ErrRateLimitExceeded) {
			httpresponse.Error(w, http.StatusTooManyRequests, "TOO_MANY_REQUESTS", "api key rate limit exceeded")
			return AuthenticatedUser{}, false
		}
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.Unauthenticated:
				httpresponse.WriteGRPCError(w, err)
				return AuthenticatedUser{}, false
			case codes.ResourceExhausted:
				httpresponse.WriteGRPCError(w, err)
				return AuthenticatedUser{}, false
			}
		}
		slog.Error("api key verification failed", "error", err)
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL", "failed to verify api key")
		return AuthenticatedUser{}, false
	}
	if user.ID == uuid.Nil {
		httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized: invalid api key")
		return AuthenticatedUser{}, false
	}

	return AuthenticatedUser{
		UserID:     user.ID,
		Role:       NormalizeRole(user.Role),
		CustomerID: user.CustomerID,
		AuthSource: "api_key",
	}, true
}

func (m *AuthMiddleware) SessionFromRequest(r *http.Request) (AuthenticatedUser, bool) {
	if m == nil || m.tokenMaker == nil {
		return AuthenticatedUser{}, false
	}
	cookie, err := r.Cookie("accessToken")
	if err != nil || cookie.Value == "" {
		return AuthenticatedUser{}, false
	}

	payload, err := m.tokenMaker.VerifyToken(cookie.Value)
	if err != nil {
		return AuthenticatedUser{}, false
	}

	if rdbs := m.controlRedis(); len(rdbs) > 0 {
		revoked, errRev := m.checkTokenRevocation(r.Context(), rdbs, payload)
		if errRev != nil || revoked {
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

func (m *AuthMiddleware) authenticate(w http.ResponseWriter, r *http.Request) (AuthenticatedUser, bool) {
	if key := r.Header.Get("X-Admin-API-Key"); key != "" && m.cfg != nil && key == string(m.cfg.AdminAPIKey) {
		return AuthenticatedUser{
			UserID:     apiKeyPrincipalID(key),
			Role:       RoleAdmin,
			CustomerID: uuid.Nil,
			AuthSource: "api_key",
			Scope:      authz.ScopeGlobal,
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

	if rdbs := m.controlRedis(); len(rdbs) > 0 {
		revoked, errRev := m.checkTokenRevocation(r.Context(), rdbs, payload)
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

func (m *AuthMiddleware) checkTokenRevocation(ctx context.Context, rdbs []redis.UniversalClient, payload *identity.Payload) (bool, error) {
	for i, rdb := range rdbs {
		if rdb == nil {
			continue
		}
		revoked, err := identity.CheckTokenRevocation(ctx, rdb, payload)
		if err != nil {
			if m.cfg != nil && m.cfg.Env == "development" {
				slog.Warn("redis revocation check failed in development, allowing session",
					"shard", i, "error", err)
				return false, nil
			}
			return false, fmt.Errorf("shard %d: %w", i, err)
		}
		if revoked {
			return true, nil
		}
	}
	return false, nil
}
