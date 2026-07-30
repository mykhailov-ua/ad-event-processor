package management

import (
	"net/http"
	"os"
	"path/filepath"

	"espx/internal/adminapi"
	"espx/internal/costsync"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

func (h *Handler) BuildAdminAPIRegistry(pool *pgxpool.Pool, rdbs []redis.UniversalClient) adminapi.RouteRegistry {
	if h == nil || h.svc == nil || pool == nil {
		return adminapi.RouteRegistry{}
	}

	limit := h.limit
	perm := h.adminRequirePermission()
	permAny := h.adminRequireAnyPermission()
	selfServePerm := h.adminSelfServePermission()
	writeErr := func(w http.ResponseWriter, err error) {
		writeServiceError(w, err)
	}
	authCustomer := h.authorizeCustomerAccess
	authCampaign := h.authorizeCampaignAccess
	isAdmin := func(r *http.Request) bool {
		u, ok := GetUser(r.Context())
		return ok && !u.IsUser()
	}

	svc := h.svc
	composite := adminapi.NewCompositeReadService(pool, h.cfg, nil)
	if composite != nil {
		composite.SetCHQuery(svc.CHQuery())
	}

	exportDir := os.Getenv("BILLING_EXPORT_PATH")
	if exportDir == "" {
		exportDir = filepath.Join(".", "data", "billing-exports")
	}
	var exportHTTP *adminapi.ExportHTTPHandlers
	if composite != nil {
		exportHTTP = &adminapi.ExportHTTPHandlers{
			JobRunner:               adminapi.NewJobRunner(composite, exportDir),
			ApplyRateLimit:          limit,
			RequirePermission:       perm,
			AuthorizeCustomerAccess: authCustomer,
			WriteServiceError:       writeErr,
		}
	}

	encKey := []byte(h.cfg.ConsentHMACSecret)
	if len(encKey) < 32 && len(h.cfg.TokenSymmetricKey) >= 32 {
		encKey = []byte(h.cfg.TokenSymmetricKey)
	}

	var costWorker *costsync.Worker
	if h.cfg != nil && h.cfg.Control.EnableCostSync && len(encKey) >= 32 {
		costWorker = costsync.NewWorker(pool, encKey)
	}

	opsReader := newOpsReader(svc)
	paymentAdapter := paymentIntentsAdapter{payment: h.payment}

	return adminapi.RouteRegistry{
		BillingHTTP: &adminapi.BillingHTTPHandlers{
			InvoiceGRPC:                  h.billing,
			CompositeReads:               composite,
			ApplyRateLimit:               limit,
			RequirePermission:            perm,
			AuthorizeCustomerAccess:      authCustomer,
			WriteServiceError:            writeErr,
			RequestIsFromAdmin:           isAdmin,
			ApplySelfServeRateLimit:      limit,
			RequireSelfServePermission:   selfServePerm,
			ResolveSelfServeCustomerID:   h.resolveSelfServeCustomerIDForBilling,
			CustomerBalance:              customerBalanceAdapter{svc: svc},
			Disputes:                     disputeListerAdapter{payment: h.payment, pool: svc},
			LimitExportByCustomer:        h.limitExportByCustomer,
			ResolveDisputeCustomerFilter: h.resolveDisputeCustomerFilter,
		},
		OpsHTTP: &adminapi.OpsHTTPHandlers{
			OpsReader:               opsReader,
			PaymentIntents:          paymentAdapter,
			ConsentRecorder:         consentRecorderAdapter{svc: svc},
			ConsentVerifier:         consentVerifierAdapter{secret: []byte(h.cfg.ConsentHMACSecret)},
			AuditLister:             auditLister{svc: svc},
			RolesReloader:           rolesReloader{mw: h.authMiddleware},
			PlansReloader:           plansReloader{pool: pool, svc: svc},
			Blacklist:               blacklistAdapter{svc: svc},
			ApplyRateLimit:          limit,
			RequirePermission:       perm,
			WriteServiceError:       writeErr,
			AuthorizeCustomerAccess: authCustomer,
		},
		ExportHTTP: exportHTTP,
		LicensingHTTP: &adminapi.LicensingHTTPHandlers{
			Pool:                       pool,
			RedisForCustomer:           func(id uuid.UUID) redis.UniversalClient { return svc.getRDB(id) },
			ApplyRateLimit:             limit,
			RequirePermission:          perm,
			ApplySelfServeRateLimit:    limit,
			RequireSelfServePermission: selfServePerm,
			ResolveSelfServeCustomerID: h.resolveSelfServeCustomerIDForBilling,
		},
		ReportsHTTP: &adminapi.ReportsHTTPHandlers{
			CampaignStats:             campaignStatsAdapter{svc: svc},
			CampaignForecaster:        campaignForecastAdapter{svc: svc},
			Pool:                      pool,
			CHQuery:                   svc.CHQuery(),
			ApplyRateLimit:            limit,
			RequirePermission:         perm,
			RequireAnyPermission:      permAny,
			AuthorizeCampaignAccess:   authCampaign,
			ResolveForecastCustomerID: h.resolveForecastCustomerID,
			WriteServiceError:         writeErr,
		},
		DashboardsHTTP: &adminapi.DashboardsHTTPHandlers{
			ApplyRateLimit:    limit,
			RequirePermission: perm,
		},
		ViewsHTTP: &adminapi.ViewsHTTPHandlers{
			Service: adminapi.NewService(),
		},
		SelfServeHTTP: &adminapi.SelfServeHTTPHandlers{
			Campaigns:                  campaignAdminAdapter{svc: svc},
			PaymentIntents:             paymentAdapter,
			Invoices:                   h.billing,
			APIKeys:                    apiKeyCreatorAdapter{auth: h.authClient},
			ApplyRateLimit:             limit,
			RequireSelfServePermission: selfServePerm,
			ResolveSelfServeCustomerID: h.resolveSelfServeCustomerIDForSelfServe,
			AuthorizeCampaignAccess:    authCampaign,
			WriteServiceError:          writeErr,
		},
		PostbackHTTP: &adminapi.PostbackHTTPHandlers{
			Pool:              pool,
			EncryptionKey:     encKey,
			ApplyRateLimit:    limit,
			RequirePermission: perm,
		},
		CostSyncHTTP: &adminapi.CostSyncHTTPHandlers{
			Pool:              pool,
			EncryptionKey:     encKey,
			Worker:            costWorker,
			ApplyRateLimit:    limit,
			RequirePermission: perm,
		},
		MarginGuardHTTP: &adminapi.MarginGuardHTTPHandlers{
			Service:           svc,
			ApplyRateLimit:    limit,
			RequirePermission: perm,
		},
		CampaignsHTTP: &adminapi.CampaignsHTTPHandlers{
			Campaigns:               campaignReaderAdapter{svc: svc},
			ApplyRateLimit:          limit,
			RequireAnyPermission:    permAny,
			AuthorizeCampaignAccess: authCampaign,
			WriteServiceError:       writeErr,
		},
	}
}

func (h *Handler) adminRequirePermission() func(string, http.HandlerFunc) http.HandlerFunc {
	return func(permission string, next http.HandlerFunc) http.HandlerFunc {
		return h.perm(next, permission)
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
	if u.IsUser() {
		if customerFilter != "" && customerFilter != u.CustomerID.String() {
			return "", errForbidden
		}
		return u.CustomerID.String(), nil
	}
	return customerFilter, nil
}
