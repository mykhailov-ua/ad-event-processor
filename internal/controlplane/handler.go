package controlplane

import (
	"context"
	"net/http"

	"espx/internal/config"
	"espx/internal/controlplane/adminapi"
	"espx/pkg/httpresponse"
)

type Handler struct {
	svc             *Service
	cfg             *config.Config
	ipLimiter       *ipRateLimiter
	customerLimiter *customerRateLimiter
	authMiddleware  *AuthMiddleware
	authClient      *AuthClient
	payment         *PaymentClient
	billing         *BillingClient
}

func NewHandler(svc *Service, cfg *config.Config, authMiddleware *AuthMiddleware, authClient *AuthClient, paymentClient *PaymentClient, billingClient *BillingClient) *Handler {
	rps := 10.0
	burst := 50
	if cfg != nil {
		rps = cfg.Management.RateLimitRPS
		burst = cfg.Management.RateLimitBurst
	}
	h := &Handler{
		svc:             svc,
		cfg:             cfg,
		ipLimiter:       newIPRateLimiter(rps, burst),
		customerLimiter: newCustomerRateLimiter(),
		authMiddleware:  authMiddleware,
		authClient:      authClient,
		payment:         paymentClient,
		billing:         billingClient,
	}
	if paymentClient != nil {
		svc.SetPayment(paymentClient)
	}
	return h
}

func (handler *Handler) RegisterRoutes(mux *http.ServeMux) {
	if handler.svc != nil && handler.svc.GetPool() != nil {
		adminapi.RegisterRoutes(mux, handler.BuildAdminAPIRegistry(handler.svc.GetPool(), handler.svc.RedisShards()))
	}

	registerAdminGoneRoutes(mux)
	registerRootRoute(mux)
	handler.registerRegionIngestRoutes(mux)
}

func (handler *Handler) limit(next http.HandlerFunc) http.HandlerFunc {
	return handler.limitByIP(handler.pgHigh(next))
}

func (handler *Handler) pgHigh(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if handler.svc == nil || handler.svc.pgGate == nil {
			next(w, r)
			return
		}
		if err := handler.svc.pgGate.AcquireHigh(r.Context()); err != nil {
			httpresponse.Error(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "database busy")
			return
		}
		defer handler.svc.pgGate.ReleaseHigh()
		next(w, r)
	}
}

func (handler *Handler) perm(next http.HandlerFunc, permission string) http.HandlerFunc {
	if handler.authMiddleware != nil {
		return handler.authMiddleware.RequirePermission(permission)(next)
	}
	return handler.authFallback(next)
}

func (handler *Handler) authFallback(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("X-Admin-API-Key")
		if key == "" || handler.cfg == nil || key != string(handler.cfg.AdminAPIKey) {
			httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
			return
		}
		user := AuthenticatedUser{
			UserID:     apiKeyPrincipalID(key),
			Role:       RoleAdmin,
			AuthSource: "api_key",
		}
		ctx := context.WithValue(r.Context(), UserContextKey, user)
		next(w, r.WithContext(ctx))
	}
}
