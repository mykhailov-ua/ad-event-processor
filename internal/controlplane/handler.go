package controlplane

import (
	"context"
	"net/http"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/controlplane/adminapi"
	"github.com/bidshard/ad-event-processor/pkg/httpresponse"
)

type Handler struct {
	svc                 *Service
	cfg                 *config.Config
	ipLimiter           *ipRateLimiter
	licenseApplyLimiter *ipRateLimiter
	customerLimiter     *customerRateLimiter
	authMiddleware      *AuthMiddleware
	authClient          *AuthClient
	payment             *PaymentClient
	billing             *BillingClient
	invoiceDelivery     adminapi.InvoiceRetryer
}

func NewHandler(svc *Service, cfg *config.Config, authMiddleware *AuthMiddleware, authClient *AuthClient, paymentClient *PaymentClient, billingClient *BillingClient) *Handler {
	rps := 10.0
	burst := 50
	if cfg != nil {
		rps = cfg.Management.RateLimitRPS
		burst = cfg.Management.RateLimitBurst
	}
	h := &Handler{
		svc:                 svc,
		cfg:                 cfg,
		ipLimiter:           newIPRateLimiter(rps, burst),
		licenseApplyLimiter: newIPRateLimiter(licenseApplyRPS, licenseApplyBurst),
		customerLimiter:     newCustomerRateLimiter(),
		authMiddleware:      authMiddleware,
		authClient:          authClient,
		payment:             paymentClient,
		billing:             billingClient,
	}
	if paymentClient != nil {
		svc.SetPayment(paymentClient)
	}
	return h
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	if h.svc != nil && h.svc.GetPool() != nil {
		adminapi.RegisterRoutes(mux, h.BuildAdminAPIRegistry(h.svc.GetPool(), h.svc.RedisShards()))
	}

	registerAdminGoneRoutes(mux)
	registerRootRoute(mux, NewAdminUIGate(h.authMiddleware))
	h.registerRegionIngestRoutes(mux)
}

func (h *Handler) limit(next http.HandlerFunc) http.HandlerFunc {
	return h.limitByIP(h.pgHigh(next))
}

func (h *Handler) pgHigh(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if h.svc == nil || h.svc.pgGate == nil {
			next(w, r)
			return
		}
		if err := h.svc.pgGate.AcquireHigh(r.Context()); err != nil {
			httpresponse.Error(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "database busy")
			return
		}
		defer h.svc.pgGate.ReleaseHigh()
		next(w, r)
	}
}

func (h *Handler) perm(next http.HandlerFunc, permission string) http.HandlerFunc {
	if h.authMiddleware != nil {
		return h.authMiddleware.RequirePermission(permission)(next)
	}
	return h.authFallback(next)
}

func (h *Handler) authFallback(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("X-Admin-API-Key")
		if key == "" || h.cfg == nil || key != string(h.cfg.AdminAPIKey) {
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
