package adminauth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"ad-event-processor/internal/campaign/selfserve"
	"ad-event-processor/internal/config"
	ctrlhttp "ad-event-processor/internal/control/http"
	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/internal/identity"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type contextKey string

const UserContextKey contextKey = "authenticated_user"

func GetUser(ctx context.Context) (authz.AuthenticatedUser, bool) {
	return authz.GetUser(ctx)
}

type Middleware struct {
	tokenMaker    identity.Maker
	redisClient   redis.UniversalClient
	controlRdbs   []redis.UniversalClient
	cfg           *config.Config
	authClient    *identity.AuthClient
	apiKeyLimiter *ctrlhttp.APIKeyRateLimiter
	policy        *authz.Store
	pool          *pgxpool.Pool
}

func New(tokenMaker identity.Maker, redisClient redis.UniversalClient, cfg *config.Config, authClient *identity.AuthClient) *Middleware {
	rps := ctrlhttp.DefaultAPIKeyRPS
	burst := ctrlhttp.DefaultAPIKeyBurst
	if cfg != nil && cfg.SelfServeAPIKeyRPS > 0 {
		rps = cfg.SelfServeAPIKeyRPS
		burst = int(rps * 2)
		if burst < 1 {
			burst = ctrlhttp.DefaultAPIKeyBurst
		}
	}
	return &Middleware{
		tokenMaker:    tokenMaker,
		redisClient:   redisClient,
		cfg:           cfg,
		authClient:    authClient,
		apiKeyLimiter: ctrlhttp.NewAPIKeyRateLimiter(rps, burst),
	}
}

func (m *Middleware) RefreshUserPolicy(userID uuid.UUID, role string) {
	if m != nil && m.policy != nil {
		m.policy.RefreshUser(userID, role)
	}
}

func (m *Middleware) SetControlRedisShards(redisShards []redis.UniversalClient) {
	m.controlRdbs = redisShards
}

func (m *Middleware) controlRedis() []redis.UniversalClient {
	if len(m.controlRdbs) > 0 {
		return m.controlRdbs
	}
	if m.redisClient != nil {
		return []redis.UniversalClient{m.redisClient}
	}
	return nil
}

func (m *Middleware) ReloadRolesYAML() error {
	if m == nil || m.policy == nil {
		return fmt.Errorf("policy store not configured")
	}
	return authz.LoadRolesYAML(authz.DefaultRolesPath(), m.policy)
}

func (m *Middleware) SetPolicyStore(store *authz.Store) {
	m.policy = store
}

func (m *Middleware) SetPool(pool *pgxpool.Pool) {
	m.pool = pool
}

func (m *Middleware) attachAuthz(ctx context.Context, user authz.AuthenticatedUser) context.Context {
	if m.policy == nil {
		ctx = authz.WithAuthenticatedUser(ctx, user)
		return context.WithValue(ctx, UserContextKey, user)
	}
	snap := m.policy.EffectivePermissionsDB(ctx, m.pool, user.UserID, user.Role)
	if len(user.APIKeyScopes) > 0 {
		snap = selfserve.RestrictSnapshotForAPIKeyScopes(snap, user.APIKeyScopes)
	}
	if user.Scope == "" {
		user.Scope = snap.Scope
	}
	if user.AuthSource == "api_key" && user.CustomerID != uuid.Nil {
		user.Scope = authz.ScopeCustomer
	}
	ctx = authz.WithSnapshot(ctx, snap)
	ctx = authz.WithAuthenticatedUser(ctx, user)
	return context.WithValue(ctx, UserContextKey, user)
}

func (m *Middleware) checkPermission(ctx context.Context, user authz.AuthenticatedUser, permission string) bool {
	if snap, ok := authz.SnapshotFromContext(ctx); ok {
		return snap.Has(permission)
	}
	return ctrlhttp.HasPermission(user.Role, permission)
}

func (m *Middleware) RequireAuthenticated() func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			user, ok := m.authenticate(w, r)
			if !ok {
				return
			}
			ctx := m.attachAuthz(r.Context(), user)
			next(w, r.WithContext(ctx))
		}
	}
}

func (m *Middleware) RequirePermission(permission string) func(http.HandlerFunc) http.HandlerFunc {
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

func (m *Middleware) RequireAnyPermission(permissions ...string) func(http.HandlerFunc) http.HandlerFunc {
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

func (m *Middleware) RequireSelfServe(permission string) func(http.HandlerFunc) http.HandlerFunc {
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

func (m *Middleware) RequireAuth(allowedRoles ...string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			user, ok := m.authenticate(w, r)
			if !ok {
				return
			}
			roleAllowed := false
			for _, allowed := range allowedRoles {
				if user.Role == ctrlhttp.NormalizeRole(allowed) || user.Role == ctrlhttp.RoleAdmin {
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

func (m *Middleware) authenticateAPIKey(w http.ResponseWriter, r *http.Request, rawKey string) (authz.AuthenticatedUser, bool) {
	if m.authClient == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "AUTH_UNAVAILABLE", "auth service not configured")
		return authz.AuthenticatedUser{}, false
	}
	if m.apiKeyLimiter != nil && !m.apiKeyLimiter.Allow(apiKeyDigest(rawKey)) {
		httpresponse.Error(w, http.StatusTooManyRequests, "TOO_MANY_REQUESTS", "api key rate limit exceeded")
		return authz.AuthenticatedUser{}, false
	}

	user, err := m.authClient.VerifyAPIKey(r.Context(), rawKey)
	if err != nil {
		if errors.Is(err, identity.ErrInvalidAPIKey) || errors.Is(err, identity.ErrInvalidCredentials) {
			httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized: invalid api key")
			return authz.AuthenticatedUser{}, false
		}
		if errors.Is(err, identity.ErrRateLimitExceeded) {
			httpresponse.Error(w, http.StatusTooManyRequests, "TOO_MANY_REQUESTS", "api key rate limit exceeded")
			return authz.AuthenticatedUser{}, false
		}
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.Unauthenticated:
				httpresponse.WriteGRPCError(w, err)
				return authz.AuthenticatedUser{}, false
			case codes.ResourceExhausted:
				httpresponse.WriteGRPCError(w, err)
				return authz.AuthenticatedUser{}, false
			}
		}
		slog.Error("api key verification failed", "error", err)
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL", "failed to verify api key")
		return authz.AuthenticatedUser{}, false
	}
	if user.ID == uuid.Nil {
		httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized: invalid api key")
		return authz.AuthenticatedUser{}, false
	}

	return authz.AuthenticatedUser{
		UserID:       user.ID,
		Role:         ctrlhttp.NormalizeRole(user.Role),
		CustomerID:   user.CustomerID,
		AuthSource:   "api_key",
		APIKeyScopes: user.Scopes,
	}, true
}

func (m *Middleware) SessionFromRequest(r *http.Request) (authz.AuthenticatedUser, bool) {
	if m == nil || m.tokenMaker == nil {
		return authz.AuthenticatedUser{}, false
	}
	cookie, err := r.Cookie("accessToken")
	if err != nil || cookie.Value == "" {
		return authz.AuthenticatedUser{}, false
	}

	payload, err := m.tokenMaker.VerifyToken(cookie.Value)
	if err != nil {
		return authz.AuthenticatedUser{}, false
	}

	if redisShards := m.controlRedis(); len(redisShards) > 0 {
		revoked, errRev := m.checkTokenRevocation(r.Context(), redisShards, payload)
		if errRev != nil || revoked {
			return authz.AuthenticatedUser{}, false
		}
	}

	return authz.AuthenticatedUser{
		UserID:     payload.UserID,
		Role:       ctrlhttp.NormalizeRole(payload.Role),
		CustomerID: payload.CustomerID,
		AuthSource: "session",
	}, true
}

func (m *Middleware) authenticate(w http.ResponseWriter, r *http.Request) (authz.AuthenticatedUser, bool) {
	if key := r.Header.Get("X-Admin-API-Key"); key != "" && m.cfg != nil && key == string(m.cfg.AdminAPIKey) {
		return authz.AuthenticatedUser{
			UserID:     apiKeyPrincipalID(key),
			Role:       ctrlhttp.RoleAdmin,
			CustomerID: uuid.Nil,
			AuthSource: "api_key",
			Scope:      authz.ScopeGlobal,
		}, true
	}

	cookie, err := r.Cookie("accessToken")
	if err != nil || cookie.Value == "" {
		httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized: missing token")
		return authz.AuthenticatedUser{}, false
	}

	payload, err := m.tokenMaker.VerifyToken(cookie.Value)
	if err != nil {
		httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized: invalid token")
		return authz.AuthenticatedUser{}, false
	}

	if redisShards := m.controlRedis(); len(redisShards) > 0 {
		revoked, errRev := m.checkTokenRevocation(r.Context(), redisShards, payload)
		if errRev != nil {
			slog.Error("redis revocation check failed, blocking request to prevent security bypass", "error", errRev)
			httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized: security check failed")
			return authz.AuthenticatedUser{}, false
		}
		if revoked {
			httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized: session revoked")
			return authz.AuthenticatedUser{}, false
		}
	}

	return authz.AuthenticatedUser{
		UserID:     payload.UserID,
		Role:       ctrlhttp.NormalizeRole(payload.Role),
		CustomerID: payload.CustomerID,
		AuthSource: "session",
	}, true
}

func (m *Middleware) checkTokenRevocation(ctx context.Context, redisShards []redis.UniversalClient, payload *identity.Payload) (bool, error) {
	for i, redisClient := range redisShards {
		if redisClient == nil {
			continue
		}
		revoked, err := identity.CheckTokenRevocation(ctx, redisClient, payload)
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

func apiKeyPrincipalID(apiKey string) uuid.UUID {
	return ctrlhttp.APIKeyPrincipalID(apiKey)
}
