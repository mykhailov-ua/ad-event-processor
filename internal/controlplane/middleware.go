package controlplane

import (
	"context"
	"net/http"

	"ad-event-processor/internal/config"
	ctrlhttp "ad-event-processor/internal/control/http"
	"ad-event-processor/internal/controlplane/adminauth"
	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/internal/costsync"
	"ad-event-processor/internal/fraudadmin"
	"ad-event-processor/internal/identity"
	"ad-event-processor/internal/platformsync"
	"ad-event-processor/internal/reportjob"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type AuthMiddleware = adminauth.Middleware

const UserContextKey = adminauth.UserContextKey

func GetUser(ctx context.Context) (authz.AuthenticatedUser, bool) {
	return authz.GetUser(ctx)
}

func NewAuthMiddleware(tokenMaker identity.Maker, redisClient redis.UniversalClient, cfg *config.Config, authClient *identity.AuthClient) *AuthMiddleware {
	return adminauth.New(tokenMaker, redisClient, cfg, authClient)
}

func (h *Handler) adminRequirePermission() func(string, http.HandlerFunc) http.HandlerFunc {
	return func(permission string, next http.HandlerFunc) http.HandlerFunc {
		return h.perm(next, permission)
	}
}

func (h *Handler) adminRequireAuth() func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		if h.authMiddleware != nil {
			return h.authMiddleware.RequireAuth(ctrlhttp.RoleAdmin, ctrlhttp.RoleManager, ctrlhttp.RoleUser, ctrlhttp.RoleBuyer, ctrlhttp.RoleSupport)(next)
		}
		return h.authFallback(next)
	}
}

func (h *Handler) adminRequireAnyPermission() func([]string, http.HandlerFunc) http.HandlerFunc {
	return func(permissions []string, next http.HandlerFunc) http.HandlerFunc {
		if h.authMiddleware != nil {
			return h.authMiddleware.RequireAnyPermission(permissions...)(next)
		}
		return h.authFallback(next)
	}
}

func (h *Handler) adminSelfServePermission() func(string, http.HandlerFunc) http.HandlerFunc {
	return func(permission string, next http.HandlerFunc) http.HandlerFunc {
		return h.selfServePerm(next, permission)
	}
}

func (h *Handler) authorizeCustomerAccess(r *http.Request, customerID string) error {
	return h.ensureCustomerAccess(r, customerID)
}

func (h *Handler) authorizeCampaignAccess(r *http.Request, campaignID uuid.UUID) error {
	return h.ensureCampaignAccess(r, campaignID)
}

func (h *Handler) resolveSelfServeCustomerIDForBilling(r *http.Request) (uuid.UUID, error) {
	return h.resolveSelfServeCustomerID(r, nil)
}

func (h *Handler) resolveSelfServeCustomerIDForSelfServe(r *http.Request, bodyCustomerID *uuid.UUID) (uuid.UUID, error) {
	return h.resolveSelfServeCustomerID(r, bodyCustomerID)
}

func (h *Handler) resolveDisputeCustomerFilter(r *http.Request) (string, error) {
	u, ok := GetUser(r.Context())
	if !ok {
		return "", errForbidden
	}
	customerFilter := r.URL.Query().Get("customer_id")
	if u.IsUser() || u.IsTeamLead() || u.IsMediaBuyer() {
		if customerFilter != "" && customerFilter != u.CustomerID.String() {
			return "", errForbidden
		}
		return u.CustomerID.String(), nil
	}
	return customerFilter, nil
}

func (h *Handler) resolveUsageExportCustomerFilter(r *http.Request, customerID, costCenter string) (string, string, error) {
	u, ok := GetUser(r.Context())
	if !ok {
		return customerID, costCenter, nil
	}
	if u.IsUser() || u.IsTeamLead() || u.IsMediaBuyer() {
		if customerID != "" && customerID != u.CustomerID.String() {
			return "", "", errForbidden
		}
		return u.CustomerID.String(), "", nil
	}
	return customerID, costCenter, nil
}

func (h *Handler) adminRequireTeamWrite() func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			u, ok := GetUser(r.Context())
			if !ok {
				writeServiceError(w, errForbidden)
				return
			}
			if u.IsMediaBuyer() {
				writeServiceError(w, errForbidden)
				return
			}
			if u.IsTeamLead() || ctrlhttp.HasPermission(u.Role, ctrlhttp.PermUsersWrite) {
				next(w, r)
				return
			}
			writeServiceError(w, errForbidden)
		}
	}
}

func (h *Handler) resolveCampaignsCustomerID(r *http.Request, bodyCustomerID *uuid.UUID) (uuid.UUID, error) {
	u, ok := GetUser(r.Context())
	if !ok {
		return uuid.Nil, errForbidden
	}
	if u.HasBoundCustomer() {
		if bodyCustomerID != nil && *bodyCustomerID != uuid.Nil && *bodyCustomerID != u.CustomerID {
			return uuid.Nil, errForbidden
		}
		return u.CustomerID, nil
	}
	if bodyCustomerID == nil || *bodyCustomerID == uuid.Nil {
		return uuid.Nil, nil
	}
	return *bodyCustomerID, nil
}

type adminWireEnv struct {
	pool                       *pgxpool.Pool
	svc                        *Service
	encKey                     []byte
	costWorker                 *costsync.Worker
	platformWorker             *platformsync.Worker
	selfServePaymentProvider   string
	selfServeCryptoSubProvider string
	fraudPresets               fraudadmin.PresetsAPI
	reportJobs                 *reportjob.ReportJobRunner
	limit                      func(http.HandlerFunc) http.HandlerFunc
	perm                       func(string, http.HandlerFunc) http.HandlerFunc
	permAny                    func([]string, http.HandlerFunc) http.HandlerFunc
	selfServePerm              func(string, http.HandlerFunc) http.HandlerFunc
	writeErr                   func(http.ResponseWriter, error)
	authCustomer               func(*http.Request, string) error
	authCampaign               func(*http.Request, uuid.UUID) error
}
