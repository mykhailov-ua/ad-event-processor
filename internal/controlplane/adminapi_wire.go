package controlplane

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"espx/internal/controlplane/adminapi"
	"espx/internal/controlplane/authz"
	"espx/internal/costsync"
	"espx/internal/edge/xdpstats"
	"espx/internal/licensing"
	"espx/internal/openrtb"
	"espx/pkg/doctor"
	"espx/pkg/platformconfig"
	"espx/pkg/supportbundle"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type consentVerifierAdapter struct{ secret []byte }

func (v consentVerifierAdapter) Verify(body []byte, signature string) error {
	return VerifyConsentHMAC(v.secret, body, signature)
}

type supportBundleWriter struct {
	pool   *pgxpool.Pool
	logDir string
}

func (w supportBundleWriter) WriteSupportBundle(ctx context.Context, out io.Writer) error {
	meta := supportbundle.Meta{}
	if diag, ok := licenseWatcherDiagnostics(); ok {
		meta.DeploymentID = diag.DeploymentID
		meta.LicenseState = string(diag.State)
		meta.DaysToExpiry = diag.DaysToExpiry
		meta.HostFingerprint = diag.HostFingerprint
		if licensing.BindModeHard(diag.BindMode) && diag.BindFingerprint != "" {
			match := diag.FingerprintMatch
			meta.FingerprintMatch = &match
		}
	} else if w.pool != nil {
		var dep uuid.UUID
		var state string
		err := w.pool.QueryRow(ctx, `
			SELECT deployment_id, state
			FROM billing.license_status
			LIMIT 1`).Scan(&dep, &state)
		if err != nil && err != pgx.ErrNoRows {
			return err
		}
		if err == nil {
			if dep != uuid.Nil {
				meta.DeploymentID = dep.String()
			}
			meta.LicenseState = state
		}
	}
	meta.HostFingerprint = licensing.HostFingerprint()
	return supportbundle.Write(ctx, out, supportbundle.Options{
		Meta:     meta,
		LogDir:   w.logDir,
		MaxBytes: supportbundle.DefaultMaxBytes,
		ExtraJSON: map[string]any{
			"client_rum.json": snapshotRUMEvents(),
		},
	})
}

type rolesReloader struct{ mw *AuthMiddleware }

func (r rolesReloader) ReloadRoles() error {
	if r.mw == nil || r.mw.policy == nil {
		return fmt.Errorf("policy store not configured")
	}
	return authz.LoadRolesYAML(authz.DefaultRolesPath(), r.mw.policy)
}

func (r rolesReloader) RolesPath() string { return authz.DefaultRolesPath() }

type platformAuthAdapter struct {
	client *AuthClient
}

func (a platformAuthAdapter) Register(ctx context.Context, adminAPIKey, email, password, role, customerID string) error {
	if a.client == nil {
		return errAuthUnavailable
	}
	_, err := a.client.Register(ctx, adminAPIKey, email, password, role, customerID)
	return err
}

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
	composite := adminapi.NewCompositeReadService(pool, h.cfg)
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
	reportJobs := adminapi.NewReportJobRunner(filepath.Join(".", "data", "report-exports"), adminapi.ReportExportDeps{
		Pool:    pool,
		CHQuery: svc.CHQuery(),
	})

	return adminapi.RouteRegistry{
		BillingHTTP: &adminapi.BillingHTTPHandlers{
			Billing:                      h.billing,
			CompositeReads:               composite,
			ApplyRateLimit:               limit,
			RequirePermission:            perm,
			AuthorizeCustomerAccess:      authCustomer,
			WriteServiceError:            writeErr,
			RequestIsFromAdmin:           isAdmin,
			ApplySelfServeRateLimit:      limit,
			RequireSelfServePermission:   selfServePerm,
			ResolveSelfServeCustomerID:   h.resolveSelfServeCustomerIDForBilling,
			CustomerBalance:              svc,
			Disputes:                     svc,
			LimitExportByCustomer:        h.limitExportByCustomer,
			ResolveDisputeCustomerFilter: h.resolveDisputeCustomerFilter,
		},
		DoctorHTTP: &adminapi.DoctorHTTPHandlers{
			Config: h.cfg,
			PlatformConfig: func(ctx context.Context) (platformconfig.Config, error) {
				cfg, _, err := svc.GetPlatformConfig(ctx)
				return cfg, err
			},
			ProbeDeps: doctor.ProbeDeps{
				Config: h.cfg,
				Redis: func(ctx context.Context) ([]redis.UniversalClient, error) {
					return svc.RedisShards(), nil
				},
				PGPool: func(ctx context.Context) (*pgxpool.Pool, error) {
					return svc.GetPool(), nil
				},
				LicenseState: licenseWatcherState,
				LicenseDiagnostics: func() (licensing.LicenseDiagnostics, bool) {
					return licenseWatcherDiagnostics()
				},
			},
			ApplyRateLimit:    limit,
			RequirePermission: perm,
			WriteServiceError: writeErr,
		},
		OpsHTTP: &adminapi.OpsHTTPHandlers{
			OpsReader:               opsReader,
			PaymentIntents:          h.payment,
			ConsentRecorder:         svc,
			ConsentVerifier:         consentVerifierAdapter{secret: []byte(h.cfg.ConsentHMACSecret)},
			AuditLister:             svc,
			RolesReloader:           rolesReloader{mw: h.authMiddleware},
			Blacklist:               svc,
			FraudThreat:             svc,
			ApplyRateLimit:          limit,
			RequirePermission:       perm,
			WriteServiceError:       writeErr,
			AuthorizeCustomerAccess: authCustomer,
			SupportBundle: supportBundleWriter{
				pool:   pool,
				logDir: h.cfg.Logger.Dir,
			},
			RUMStore: rumStoreAdapter{},
		},
		ExportHTTP: exportHTTP,
		LicensingHTTP: &adminapi.LicensingHTTPHandlers{
			Pool:                  pool,
			LicenseService:        svc,
			ApplyRateLimit:        limit,
			LicenseApplyRateLimit: h.limitLicenseApply,
			RequirePermission:     perm,
			WriteServiceError:     writeErr,
		},
		ReportsHTTP: &adminapi.ReportsHTTPHandlers{
			CampaignStats:             svc,
			CampaignForecaster:        svc,
			ReportJobs:                reportJobs,
			Pool:                      pool,
			CHQuery:                   svc.CHQuery(),
			ApplyRateLimit:            limit,
			RequirePermission:         perm,
			RequireAnyPermission:      permAny,
			AuthorizeCampaignAccess:   authCampaign,
			AuthorizeCustomerAccess:   authCustomer,
			ResolveForecastCustomerID: h.resolveForecastCustomerID,
			WriteServiceError:         writeErr,
		},
		DashboardsHTTP: &adminapi.DashboardsHTTPHandlers{
			BuyerPortfolio:       svc,
			CampaignDashboard:    svc,
			RoleDashboards:       svc,
			ReportJobs:           reportJobs,
			ApplyRateLimit:       limit,
			RequirePermission:    perm,
			RequireAnyPermission: permAny,
			ResolveCustomerID:    h.resolveCampaignsCustomerID,
			WriteServiceError:    writeErr,
			EdgeMetricsReader:    FetchEdgeMetrics,
			XDPStatsReader: func(ctx context.Context) (xdpstats.Snapshot, error) {
				return xdpstats.ReadRedisAny(ctx, svc.RedisShards())
			},
		},
		ViewsHTTP: &adminapi.ViewsHTTPHandlers{
			Service:                 adminapi.NewService(),
			ApplyRateLimit:          limit,
			RequirePermission:       perm,
			RequireAnyPermission:    permAny,
			AuthorizeCustomerAccess: authCustomer,
			WriteServiceError:       writeErr,
		},
		SelfServeHTTP: &adminapi.SelfServeHTTPHandlers{
			Campaigns:                  svc,
			PaymentIntents:             h.payment,
			Invoices:                   h.billing,
			APIKeys:                    h.authClient,
			ApplyRateLimit:             limit,
			RequireSelfServePermission: selfServePerm,
			RequireAnyPermission:       permAny,
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
		RtbFloorsHTTP: &adminapi.RtbFloorsHTTPHandlers{
			Service:           svc,
			ApplyRateLimit:    limit,
			RequirePermission: perm,
			WriteServiceError: writeErr,
		},
		RtbHTTP: &adminapi.RtbHTTPHandlers{
			Service:           &rtbAdminService{svc: svc},
			ApplyRateLimit:    limit,
			RequirePermission: perm,
			WriteServiceError: writeErr,
			ExchangeConfig: openrtb.ExchangeConfig{
				NoBidMode:   h.cfg.RtbExchangeNoBidMode,
				MultiImpMax: h.cfg.RtbExchangeMultiImpMax,
				RegsPolicy:  h.cfg.RtbRegsPolicy,
				Blocklist:   h.cfg.RtbBlocklistEnforce,
			},
			ReconcileCH: func(ctx context.Context, requestID string, window time.Duration) (uint64, uint64, int64, bool) {
				stats, ok := svc.RtbReconcileCHStats(ctx, requestID, window)
				if !ok {
					return 0, 0, 0, false
				}
				return stats.Bids, stats.Wins, stats.SpendMicro, true
			},
		},
		CampaignsHTTP: &adminapi.CampaignsHTTPHandlers{
			Campaigns:               svc,
			ApplyRateLimit:          limit,
			RequireAnyPermission:    permAny,
			AuthorizeCampaignAccess: authCampaign,
			ResolveCustomerID:       h.resolveCampaignsCustomerID,
			WriteServiceError:       writeErr,
		},
		CustomersHTTP: &adminapi.CustomersHTTPHandlers{
			Customers:               svc,
			ApplyRateLimit:          limit,
			RequirePermission:       perm,
			AuthorizeCustomerAccess: authCustomer,
			WriteServiceError:       writeErr,
		},
		SupportHTTP: &adminapi.SupportHTTPHandlers{
			Feedback: svc,
			SupportBundle: supportBundleWriter{
				pool:   pool,
				logDir: h.cfg.Logger.Dir,
			},
			ApplyRateLimit:    limit,
			RequireAuth:       h.adminRequireAuth(),
			WriteServiceError: writeErr,
		},
		MetaHTTP: &adminapi.MetaHTTPHandlers{
			ApplyRateLimit: limit,
			Enrich:         h.metaEnricher(),
			WriteError:     writeErr,
		},
		EulaHTTP: &adminapi.EulaHTTPHandlers{
			Service:           svc,
			ApplyRateLimit:    limit,
			RequirePermission: perm,
			WriteServiceError: writeErr,
		},
		PlatformHTTP: &adminapi.PlatformHTTPHandlers{
			Service:           svc,
			AuthClient:        platformAuthAdapter{client: h.authClient},
			Cfg:               h.cfg,
			ApplyRateLimit:    limit,
			RequirePermission: perm,
			WriteServiceError: writeErr,
		},
		StubHTTP: &adminapi.StubHTTPHandlers{
			ApplyRateLimit:    limit,
			RequirePermission: perm,
		},
		TelegramHTTP: &adminapi.TelegramHTTPHandlers{
			Telegram:             NewTelegramService(svc, pool, rdbs),
			ApplyRateLimit:       limit,
			RequireAnyPermission: permAny,
			WriteServiceError:    writeErr,
		},
	}
}

func (h *Handler) adminRequirePermission() func(string, http.HandlerFunc) http.HandlerFunc {
	return func(permission string, next http.HandlerFunc) http.HandlerFunc {
		return h.perm(next, permission)
	}
}

func (h *Handler) adminRequireAuth() func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		if h.authMiddleware != nil {
			return h.authMiddleware.RequireAuth(RoleAdmin, RoleManager, RoleUser, RoleBuyer, RoleSupport)(next)
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
	if u.IsUser() {
		if customerFilter != "" && customerFilter != u.CustomerID.String() {
			return "", errForbidden
		}
		return u.CustomerID.String(), nil
	}
	return customerFilter, nil
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
