package controlplane

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/internal/costsync"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/edge"
	"ad-event-processor/internal/licensing"
	"ad-event-processor/internal/openrtb"
	"ad-event-processor/internal/payment"
	"ad-event-processor/internal/platformsync"
	"ad-event-processor/pkg/doctor"
	"ad-event-processor/pkg/platformconfig"
	"ad-event-processor/pkg/supportbundle"

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
		meta.HostHWIDv2 = diag.HostHWID
		if licensing.BindModeHard(diag.BindMode) && (diag.BindFingerprint != "" || diag.BindHWIDHash != "") {
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
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
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
	meta.HostHWIDv2 = licensing.HostHWID()
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

func (h *Handler) BuildAdminAPIRegistry(pool *pgxpool.Pool, redisShards []redis.UniversalClient) RouteRegistry {
	if h == nil || h.svc == nil || pool == nil {
		return RouteRegistry{}
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
	composite := NewCompositeReadService(pool, h.cfg)
	if composite != nil {
		composite.SetCHQuery(svc.CHQuery())
	}

	exportDir := os.Getenv("BILLING_EXPORT_PATH")
	if exportDir == "" {
		exportDir = filepath.Join(".", "data", "billing-exports")
	}
	var exportHTTP *ExportHTTPHandlers
	if composite != nil {
		jobRunner := NewJobRunner(composite, exportDir)
		if h.cfg != nil {
			jobRunner.ConfigureExport(int32(h.cfg.Billing.ExportFetchRows), time.Duration(h.cfg.Billing.ExportJobTimeoutMin)*time.Minute)
		}
		exportHTTP = &ExportHTTPHandlers{
			JobRunner:               jobRunner,
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
	var platformWorker *platformsync.Worker
	if h.cfg != nil && h.cfg.Control.EnableCostSync && len(encKey) >= 32 {
		costWorker = costsync.NewWorker(pool, encKey)
		platformWorker = platformsync.NewWorker(pool, encKey, costWorker)
	}

	selfServePaymentProvider := ""
	selfServeCryptoSubProvider := ""
	if h.cfg != nil {
		if h.cfg.Billing.PaymentProvider == "crypto" || payment.CryptoConfigured(h.cfg) {
			selfServePaymentProvider = "crypto"
		}
		if string(h.cfg.BTCPayWebhookSecret) != "" {
			selfServeCryptoSubProvider = payment.CryptoProviderBTCPay
		} else if string(h.cfg.CryptomusAPIKey) != "" {
			selfServeCryptoSubProvider = payment.CryptoProviderCryptomus
		}
	}

	opsReader := newOpsReader(svc)
	fraudPresets := fraudPresetsAPIAdapter{svc: svc}
	reportJobs := h.svc.ReportJobRunner()
	if reportJobs == nil {
		reportJobs = h.svc.InitReportJobRunner(reportExportDirFromWire())
	}

	return RouteRegistry{
		BillingHTTP: &BillingHTTPHandlers{
			Billing:                          h.billing,
			InvoiceDelivery:                  h.invoiceDelivery,
			CompositeReads:                   composite,
			ApplyRateLimit:                   limit,
			RequirePermission:                perm,
			AuthorizeCustomerAccess:          authCustomer,
			WriteServiceError:                writeErr,
			RequestIsFromAdmin:               isAdmin,
			ApplySelfServeRateLimit:          limit,
			RequireSelfServePermission:       selfServePerm,
			ResolveSelfServeCustomerID:       h.resolveSelfServeCustomerIDForBilling,
			CustomerBalance:                  svc,
			UsageExport:                      svc,
			ResolveUsageExportCustomerFilter: h.resolveUsageExportCustomerFilter,
			Disputes:                         svc,
			LimitExportByCustomer:            h.limitExportByCustomer,
			ResolveDisputeCustomerFilter:     h.resolveDisputeCustomerFilter,
		},
		CryptoBillingWebhook: &CryptoBillingWebhookHandlers{
			Processor:           svc,
			CryptoWebhookSecret: string(h.cfg.CryptoWebhookSecret),
			BTCPayWebhookSecret: string(h.cfg.BTCPayWebhookSecret),
			CryptomusAPIKey:     string(h.cfg.CryptomusAPIKey),
		},
		DoctorHTTP: &DoctorHTTPHandlers{
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
				LicenseState:       licenseWatcherState,
				LicenseDiagnostics: licenseWatcherDiagnostics,
				XDPStatsReader: func(ctx context.Context) (edge.Snapshot, error) {
					return edge.ReadRedisAny(ctx, svc.RedisShards())
				},
			},
			ApplyRateLimit:    limit,
			RequirePermission: perm,
			WriteServiceError: writeErr,
		},
		OpsHTTP: &OpsHTTPHandlers{
			OpsReader:               opsReader,
			PaymentIntents:          h.payment,
			ConsentRecorder:         svc,
			ConsentVerifier:         consentVerifierAdapter{secret: []byte(h.cfg.ConsentHMACSecret)},
			AuditLister:             svc,
			RolesReloader:           rolesReloader{mw: h.authMiddleware},
			Blacklist:               svc,
			Shard0Catchup:           svc,
			FraudThreat:             svc,
			ApplyRateLimit:          limit,
			RequirePermission:       perm,
			WriteServiceError:       writeErr,
			AuthorizeCustomerAccess: authCustomer,
			SupportBundle: supportBundleWriter{
				pool:   pool,
				logDir: h.cfg.Logger.Dir,
			},
			RUMStore:     rumStoreAdapter{},
			FraudPresets: fraudPresets,
		},
		ExportHTTP: exportHTTP,
		LicensingHTTP: &LicensingHTTPHandlers{
			Pool:                  pool,
			LicenseService:        svc,
			LicenseDiagnostics:    licenseWatcherDiagnostics,
			ApplyRateLimit:        limit,
			LicenseApplyRateLimit: h.limitLicenseApply,
			RequirePermission:     perm,
			WriteServiceError:     writeErr,
		},
		ReportsHTTP: &ReportsHTTPHandlers{
			CampaignStats:             svc,
			CampaignForecaster:        svc,
			ReportJobs:                reportJobs,
			Pool:                      pool,
			CHQuery:                   svc.CHQuery(),
			BuyerPortfolio:            svc,
			EdgeMetricsReader:         FetchEdgeMetrics,
			ApplyRateLimit:            limit,
			RequirePermission:         perm,
			RequireAnyPermission:      permAny,
			AuthorizeCampaignAccess:   authCampaign,
			AuthorizeCustomerAccess:   authCustomer,
			ResolveForecastCustomerID: h.resolveForecastCustomerID,
			WriteServiceError:         writeErr,
		},
		DashboardsHTTP: &DashboardsHTTPHandlers{
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
			XDPStatsReader: func(ctx context.Context) (edge.Snapshot, error) {
				return edge.ReadRedisAny(ctx, svc.RedisShards())
			},
		},
		ViewsHTTP: &ViewsHTTPHandlers{
			Store:                   NewViewsStore(pool),
			ApplyRateLimit:          limit,
			RequirePermission:       perm,
			RequireAnyPermission:    permAny,
			AuthorizeCustomerAccess: authCustomer,
			WriteServiceError:       writeErr,
		},
		SelfServeHTTP: &SelfServeHTTPHandlers{
			Campaigns:                  svc,
			Templates:                  selfServeTemplatesAdapter{svc: svc},
			PaymentIntents:             h.payment,
			Invoices:                   h.billing,
			APIKeys:                    h.authClient,
			ApplyRateLimit:             limit,
			RequireSelfServePermission: selfServePerm,
			RequireAnyPermission:       permAny,
			ResolveSelfServeCustomerID: h.resolveSelfServeCustomerIDForSelfServe,
			AuthorizeCampaignAccess:    authCampaign,
			WriteServiceError:          writeErr,
			DefaultPaymentProvider:     selfServePaymentProvider,
			CryptoSubProvider:          selfServeCryptoSubProvider,
		},
		PostbackHTTP: &PostbackHTTPHandlers{
			Pool:              pool,
			EncryptionKey:     encKey,
			ApplyRateLimit:    limit,
			RequirePermission: perm,
		},
		CostSyncHTTP: &CostSyncHTTPHandlers{
			Pool:              pool,
			EncryptionKey:     encKey,
			Worker:            costWorker,
			ApplyRateLimit:    limit,
			RequirePermission: perm,
		},
		PlatformCampaignHTTP: &PlatformCampaignHTTPHandlers{
			Pool:              pool,
			Worker:            platformWorker,
			ApplyRateLimit:    limit,
			RequirePermission: perm,
			Audit: func(ctx context.Context, q db.Querier, adminID uuid.UUID, action, targetType string, targetID *uuid.UUID, changes, metadata any) {
				svc.AuditLog(ctx, q, adminID, action, targetType, targetID, changes, metadata)
			},
			ResolveActorID: func(r *http.Request) uuid.UUID {
				u, ok := GetUser(r.Context())
				if !ok {
					return uuid.Nil
				}
				return u.UserID
			},
		},
		MarginGuardHTTP: &MarginGuardHTTPHandlers{
			Service:           svc,
			ApplyRateLimit:    limit,
			RequirePermission: perm,
		},
		SmartAlertsHTTP: &SmartAlertsHTTPHandlers{
			Service:           svc,
			ApplyRateLimit:    limit,
			RequirePermission: perm,
			ResolveActorID: func(ctx context.Context) uuid.UUID {
				u, ok := GetUser(ctx)
				if !ok {
					return uuid.Nil
				}
				return u.UserID
			},
		},
		AutomationHTTP: &AutomationHTTPHandlers{
			Service:           svc,
			ApplyRateLimit:    limit,
			RequirePermission: perm,
		},
		DomainHealthHTTP: &DomainHealthHTTPHandlers{
			Service:           svc,
			ApplyRateLimit:    limit,
			RequirePermission: perm,
			TLSAskToken:       string(h.cfg.Management.CaddyTLSAskToken),
			TLSAskAllowLocal:  h.cfg.Management.CaddyTLSAskAllowLocal,
		},
		FlowHTTP: &FlowHTTPHandlers{
			Service:           svc,
			ApplyRateLimit:    limit,
			RequirePermission: perm,
		},
		IntegrationSchemaHTTP: &IntegrationSchemaHTTPHandlers{
			Pool:              pool,
			EncryptionKey:     encKey,
			TemplateCatalog:   svc,
			ApplyRateLimit:    limit,
			RequirePermission: perm,
			ResolveTrackingDomain: func(ctx context.Context) string {
				cfg, _, err := svc.GetPlatformConfig(ctx)
				if err != nil {
					return ""
				}
				return cfg.TrackingDomain
			},
		},
		TeamHTTP: &TeamHTTPHandlers{
			Team:                 &TeamOverviewService{Pool: pool},
			Governance:           svc,
			ApplyRateLimit:       limit,
			RequireAnyPermission: permAny,
			RequireTeamWrite:     h.adminRequireTeamWrite(),
			ResolveCustomerID:    h.resolveCampaignsCustomerID,
			SnapshotFromRequest: func(r *http.Request) (authz.Snapshot, bool) {
				return authz.SnapshotFromContext(r.Context())
			},
			ActorUserID: func(r *http.Request) (uuid.UUID, bool) {
				u, ok := GetUser(r.Context())
				return u.UserID, ok
			},
			WriteServiceError: writeErr,
		},
		PublisherHTTP: &PublisherHTTPHandlers{
			Publisher:            publisherAdminAdapter{svc: svc},
			ApplyRateLimit:       limit,
			RequireAnyPermission: permAny,
			ActorUserID: func(r *http.Request) (uuid.UUID, bool) {
				u, ok := GetUser(r.Context())
				return u.UserID, ok
			},
			WriteServiceError: writeErr,
		},
		CommercialHTTP: &CommercialHTTPHandlers{
			Commercial:              commercialAdminAdapter{svc: svc},
			ApplyRateLimit:          limit,
			RequirePermission:       perm,
			RequireAnyPermission:    permAny,
			AuthorizeCustomerAccess: authCustomer,
			WriteServiceError:       writeErr,
		},
		RtbFloorsHTTP: &RtbFloorsHTTPHandlers{
			Service:           svc,
			ApplyRateLimit:    limit,
			RequirePermission: perm,
			WriteServiceError: writeErr,
		},
		RtbHTTP: &RtbHTTPHandlers{
			Service:           &rtbAdminService{svc: svc},
			ApplyRateLimit:    limit,
			RequirePermission: perm,
			WriteServiceError: writeErr,
			RuntimeConfig:     rtbRuntimeConfig{cfg: h.cfg},
			PlatformConfig: func(ctx context.Context) (platformconfig.Config, error) {
				cfg, _, err := svc.GetPlatformConfig(ctx)
				return cfg, err
			},
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
		CampaignsHTTP: &CampaignsHTTPHandlers{
			Campaigns:               svc,
			CampaignFraud:           campaignFraudAPIAdapter{svc: svc},
			ConversionMappings:      svc,
			ApplyRateLimit:          limit,
			RequireAnyPermission:    permAny,
			AuthorizeCampaignAccess: authCampaign,
			ResolveCustomerID:       h.resolveCampaignsCustomerID,
			AllowFraudPreview:       h.allowFraudPreview,
			WriteServiceError:       writeErr,
		},
		FraudHTTP: &FraudHTTPHandlers{
			Labels:                  fraudLabelsAPIAdapter{svc: svc},
			Decisions:               fraudDecisionsAPIAdapter{svc: svc},
			Integrations:            fraudIntegrationsAPIAdapter{svc: svc},
			Overrides:               fraudOverridesAPIAdapter{svc: svc},
			Presets:                 fraudPresets,
			ApplyRateLimit:          limit,
			AllowFraudDecision:      h.allowFraudDecision,
			RequirePermission:       perm,
			RequireAnyPermission:    permAny,
			ResolveCustomerID:       h.resolveCampaignsCustomerID,
			AuthorizeCampaignAccess: authCampaign,
			WriteServiceError:       writeErr,
		},
		CustomersHTTP: &CustomersHTTPHandlers{
			Customers:               svc,
			CostCenter:              svc,
			ApplyRateLimit:          limit,
			RequirePermission:       perm,
			AuthorizeCustomerAccess: authCustomer,
			WriteServiceError:       writeErr,
		},
		SupportHTTP: &SupportHTTPHandlers{
			Feedback: svc,
			SupportBundle: supportBundleWriter{
				pool:   pool,
				logDir: h.cfg.Logger.Dir,
			},
			ApplyRateLimit:    limit,
			RequireAuth:       h.adminRequireAuth(),
			WriteServiceError: writeErr,
		},
		MetaHTTP: &MetaHTTPHandlers{
			ApplyRateLimit: limit,
			Enrich:         h.metaEnricher(),
			WriteError:     writeErr,
		},
		EulaHTTP: &EulaHTTPHandlers{
			Service:           svc,
			ApplyRateLimit:    limit,
			RequirePermission: perm,
			WriteServiceError: writeErr,
		},
		PlatformHTTP: &PlatformHTTPHandlers{
			Service:           svc,
			AuthClient:        platformAuthAdapter{client: h.authClient},
			Cfg:               h.cfg,
			ApplyRateLimit:    limit,
			RequirePermission: perm,
			WriteServiceError: writeErr,
		},
		StubHTTP: &StubHTTPHandlers{
			ApplyRateLimit:    limit,
			RequirePermission: perm,
		},
		TelegramHTTP: &TelegramHTTPHandlers{
			Telegram:             NewTelegramService(svc, pool, redisShards),
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
			if u.IsTeamLead() || HasPermission(u.Role, PermUsersWrite) {
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
