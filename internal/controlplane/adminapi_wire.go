package controlplane

import (
	ctrlhttp "ad-event-processor/internal/control/http"
	"ad-event-processor/internal/telegram"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"ad-event-processor/internal/automation"
	"ad-event-processor/internal/billingadmin"
	"ad-event-processor/internal/brand"
	"ad-event-processor/internal/campaign"
	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/internal/costsync"
	"ad-event-processor/internal/dashboardadmin"
	"ad-event-processor/internal/doctor"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/edge"
	"ad-event-processor/internal/flow"
	"ad-event-processor/internal/fraudadmin"
	"ad-event-processor/internal/licensing"
	"ad-event-processor/internal/licensingadmin"
	"ad-event-processor/internal/marginguard"
	"ad-event-processor/internal/openrtb"
	"ad-event-processor/internal/opsadmin"
	"ad-event-processor/internal/payment"
	"ad-event-processor/internal/platformadmin"
	"ad-event-processor/internal/platformsync"
	"ad-event-processor/internal/reportjob"
	"ad-event-processor/internal/reports"
	"ad-event-processor/internal/rtbadmin"
	"ad-event-processor/internal/supply"
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
			"client_rum.json": opsadmin.SnapshotRUMEvents(),
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
	composite := billingadmin.NewCompositeReadService(pool, h.cfg)
	if composite != nil {
		composite.SetClickHouseQuery(svc.ClickHouseQuery())
	}

	exportDir := os.Getenv("BILLING_EXPORT_PATH")
	if exportDir == "" {
		exportDir = filepath.Join(".", "data", "billing-exports")
	}
	var exportHTTP *billingadmin.ExportHTTPHandlers
	if composite != nil {
		jobRunner := billingadmin.NewJobRunner(composite, exportDir)
		if h.cfg != nil {
			jobRunner.ConfigureExport(int32(h.cfg.Billing.ExportFetchRows), time.Duration(h.cfg.Billing.ExportJobTimeoutMin)*time.Minute)
		}
		exportHTTP = &billingadmin.ExportHTTPHandlers{
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
		BillingHTTP: &billingadmin.HTTPHandlers{
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
		CryptoBillingWebhook: &billingadmin.CryptoWebhookHandlers{
			Processor:           svc,
			CryptoWebhookSecret: string(h.cfg.CryptoWebhookSecret),
			BTCPayWebhookSecret: string(h.cfg.BTCPayWebhookSecret),
			CryptomusAPIKey:     string(h.cfg.CryptomusAPIKey),
		},
		DoctorHTTP: &doctor.DoctorHTTPHandlers{
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
		OpsHTTP: &opsadmin.HTTPHandlers{
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
			RUMStore:     opsadmin.NewRUMStoreAdapter(),
			FraudPresets: fraudPresets,
		},
		ExportHTTP: exportHTTP,
		LicensingHTTP: &licensingadmin.HTTPHandlers{
			Pool:                  pool,
			LicenseService:        svc,
			LicenseDiagnostics:    licenseWatcherDiagnostics,
			ApplyRateLimit:        limit,
			LicenseApplyRateLimit: h.limitLicenseApply,
			RequirePermission:     perm,
			WriteServiceError:     writeErr,
		},
		ReportsHTTP: &reports.ReportsHTTPHandlers{
			CampaignStats:      campaignStatsAdapter{svc: svc},
			CampaignForecaster: campaignForecasterAdapter{svc: svc},
			Pool:               pool,
			ClickHouseQuery:    svc.ClickHouseQuery(),
			BuyerPortfolio:     buyerPortfolioAdapter{svc: svc},
			EdgeMetricsReader: func(ctx context.Context) (reports.EdgeMetricsPanelDTO, error) {
				dto, err := opsadmin.FetchEdgeMetrics(ctx)
				if err != nil {
					return reports.EdgeMetricsPanelDTO{}, err
				}
				return reports.EdgeMetricsPanelDTO{
					UpdatedAt: dto.UpdatedAt, IngressH1: dto.IngressH1, IngressH2: dto.IngressH2, IngressH3: dto.IngressH3,
					BodyStream: dto.BodyStream, BodyPeek: dto.BodyPeek, BodyRead: dto.BodyRead,
					Blocked: dto.Blocked, TarpitTotal: dto.TarpitTotal, BlacklistStale: dto.BlacklistStale,
				}, nil
			},
			ApplyRateLimit:            limit,
			RequirePermission:         perm,
			RequireAnyPermission:      permAny,
			AuthorizeCampaignAccess:   authCampaign,
			AuthorizeCustomerAccess:   authCustomer,
			ResolveForecastCustomerID: h.resolveForecastCustomerID,
			WriteServiceError:         writeErr,
			RequestHasShardsRead:      requestHasShardsRead,
			RequireLicenseFeature:     requireLicenseFeature,
			DenyScopedAPIKeyReport:    campaign.DenyScopedAPIKeyOperatorReport,
		},
		ReportJobHTTP: &reportjob.HTTPHandlers{
			Runner:                  reportJobs,
			Pool:                    pool,
			ApplyRateLimit:          limit,
			RequirePermission:       perm,
			RequireAnyPermission:    permAny,
			AuthorizeCustomerAccess: authCustomer,
			ValidateReportSchedule:  reports.ValidateReportScheduleForActor,
			WriteServiceError:       writeErr,
		},
		DashboardsHTTP: &dashboardadmin.HTTPHandlers{
			BuyerPortfolio:       svc,
			CampaignDashboard:    svc,
			RoleDashboards:       svc,
			ReportJobs:           reportJobs,
			ApplyRateLimit:       limit,
			RequirePermission:    perm,
			RequireAnyPermission: permAny,
			ResolveCustomerID:    h.resolveCampaignsCustomerID,
			WriteServiceError:    writeErr,
			EdgeMetricsReader: func(ctx context.Context) (dashboardadmin.EdgeMetricsPanelDTO, error) {
				panel, err := opsadmin.FetchEdgeMetrics(ctx)
				if err != nil {
					return dashboardadmin.EdgeMetricsPanelDTO{}, err
				}
				return dashboardadmin.EdgeMetricsPanelDTO{
					UpdatedAt: panel.UpdatedAt, IngressH1: panel.IngressH1, IngressH2: panel.IngressH2, IngressH3: panel.IngressH3,
					BodyStream: panel.BodyStream, BodyPeek: panel.BodyPeek, BodyRead: panel.BodyRead,
					Blocked: panel.Blocked, TarpitTotal: panel.TarpitTotal, BlacklistStale: panel.BlacklistStale,
				}, nil
			},
			XDPStatsReader: func(ctx context.Context) (edge.Snapshot, error) {
				return edge.ReadRedisAny(ctx, svc.RedisShards())
			},
		},
		ViewsHTTP: &reports.ViewsHTTPHandlers{
			Store:                   reports.NewViewsStore(pool),
			ApplyRateLimit:          limit,
			RequirePermission:       perm,
			RequireAnyPermission:    permAny,
			AuthorizeCustomerAccess: authCustomer,
			WriteServiceError:       writeErr,
		},
		SelfServeHTTP: &campaign.SelfServeHTTPHandlers{
			Campaigns:                  svc,
			Templates:                  campaign.NewSelfServeTemplatesAdapter(svc),
			PaymentIntents:             h.payment,
			Invoices:                   h.billing,
			APIKeys:                    h.authClient,
			ApplyRateLimit:             limit,
			RequireSelfServePermission: selfServePerm,
			RequireAnyPermission:       permAny,
			ResolveSelfServeCustomerID: h.resolveSelfServeCustomerIDForSelfServe,
			AuthorizeCampaignAccess:    authCampaign,
			WriteServiceError:          writeErr,
			WriteBillingError:          billingadmin.WriteBillingError,
			DefaultPaymentProvider:     selfServePaymentProvider,
			CryptoSubProvider:          selfServeCryptoSubProvider,
		},
		PostbackHTTP: &campaign.PostbackHTTPHandlers{
			Pool:              pool,
			EncryptionKey:     encKey,
			ApplyRateLimit:    limit,
			RequirePermission: perm,
		},
		CostSyncHTTP: &billingadmin.CostSyncHTTPHandlers{
			Pool:              pool,
			EncryptionKey:     encKey,
			Worker:            costWorker,
			ApplyRateLimit:    limit,
			RequirePermission: perm,
		},
		PlatformCampaignHTTP: &platformadmin.PlatformCampaignHTTPHandlers{
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
		MarginGuardHTTP: &marginguard.HTTPHandlers{
			Service:           marginGuardServiceAdapter{svc: svc},
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
		AutomationHTTP: &automation.HTTPHandlers{
			Rules:             svc.AutomationRules(),
			ApplyRateLimit:    limit,
			RequirePermission: perm,
		},
		DomainHealthHTTP: &platformadmin.DomainHealthHTTPHandlers{
			Service:           svc,
			ApplyRateLimit:    limit,
			RequirePermission: perm,
			TLSAskToken:       string(h.cfg.Management.CaddyTLSAskToken),
			TLSAskAllowLocal:  h.cfg.Management.CaddyTLSAskAllowLocal,
		},
		FlowHTTP: &flow.HTTPHandlers{
			Service:           svc,
			ApplyRateLimit:    limit,
			RequirePermission: perm,
		},
		IntegrationSchemaHTTP: &campaign.IntegrationSchemaHTTPHandlers{
			Pool:              pool,
			EncryptionKey:     encKey,
			TemplateCatalog:   svc.TemplateCatalog(pool),
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
		TeamHTTP: &platformadmin.TeamHTTPHandlers{
			Team:                 &platformadmin.TeamOverviewService{Pool: pool},
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
		PublisherHTTP: &dashboardadmin.PublisherHTTPHandlers{
			Publisher:            svc,
			ApplyRateLimit:       limit,
			RequireAnyPermission: permAny,
			ActorUserID: func(r *http.Request) (uuid.UUID, bool) {
				u, ok := GetUser(r.Context())
				return u.UserID, ok
			},
			WriteServiceError: writeErr,
		},
		BrandHTTP: &brand.HTTPHandlers{
			Admin:                   brand.NewAdminAdapter(svc),
			ApplyRateLimit:          limit,
			RequirePermission:       perm,
			AuthorizeCustomerAccess: authCustomer,
			WriteServiceError:       writeErr,
		},
		SupplyHTTP: &supply.HTTPHandlers{
			Admin:                supply.NewAdminAdapter(supplyAdminHost{svc: svc}),
			ApplyRateLimit:       limit,
			RequirePermission:    perm,
			RequireAnyPermission: permAny,
			WriteServiceError:    writeErr,
		},
		RtbFloorsHTTP: &rtbadmin.FloorsHTTPHandlers{
			Service:           svc,
			ApplyRateLimit:    limit,
			RequirePermission: perm,
			WriteServiceError: writeErr,
		},
		RtbHTTP: &rtbadmin.HTTPHandlers{
			Service:           svc.RtbAdminService(),
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
		CampaignsHTTP: &campaign.CampaignsHTTPHandlers{
			Campaigns:                 svc.CampaignRuntime(),
			CampaignFraud:             campaignFraudAPIAdapter{svc: svc},
			ConversionMappings:        svc,
			GetCampaignFlow:           svc.GetFlow,
			ValidateCampaignFlowPaths: svc.ValidateCampaignFlowPaths,
			RecordRevisionConflict:    svc.AuditCampaignRevisionConflict,
			ClickHouseQuery:           svc.ClickHouseQuery(),
			ApplyRateLimit:            limit,
			RequireAnyPermission:      permAny,
			AuthorizeCampaignAccess:   authCampaign,
			ResolveCustomerID:         h.resolveCampaignsCustomerID,
			AllowFraudPreview:         h.allowFraudPreview,
			LicenseFeatureAllowed:     licenseFeatureAllowed,
			ReportJobs:                reportJobs,
			WriteServiceError:         writeErr,
		},
		FraudHTTP: &fraudadmin.HTTPHandlers{
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
		CustomersHTTP: &platformadmin.CustomersHTTPHandlers{
			Customers:               svc,
			CostCenter:              svc,
			ApplyRateLimit:          limit,
			RequirePermission:       perm,
			AuthorizeCustomerAccess: authCustomer,
			WriteServiceError:       writeErr,
		},
		SupportHTTP: &platformadmin.SupportHTTPHandlers{
			Feedback: svc,
			SupportBundle: supportBundleWriter{
				pool:   pool,
				logDir: h.cfg.Logger.Dir,
			},
			ApplyRateLimit:    limit,
			RequireAuth:       h.adminRequireAuth(),
			WriteServiceError: writeErr,
		},
		MetaHTTP: &platformadmin.MetaHTTPHandlers{
			ApplyRateLimit: limit,
			Enrich:         platformadmin.NewMetaEnricher(h.svc),
			WriteError:     writeErr,
		},
		SessionHTTP: func() *platformadmin.SessionHTTPHandlers {
			sh := wireSessionHTTPHandlers(func(ctx context.Context) reports.DataFreshnessDTO {
				if h.svc != nil && h.svc.clickhouseQuery != nil {
					lag, _ := h.svc.clickHouseIngestionLag(ctx)
					return portfolioFreshness(time.Now().UTC(), true, lag)
				}
				return reports.DataFreshnessDTO{Consistency: "eventual"}
			})
			sh.ApplyRateLimit = limit
			return sh
		}(),
		EulaHTTP: &licensingadmin.EulaHTTPHandlers{
			Service:           svc,
			ApplyRateLimit:    limit,
			RequirePermission: perm,
			WriteServiceError: writeErr,
		},
		PlatformHTTP: &platformadmin.HTTPHandlers{
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
		TelegramHTTP: &telegram.HTTPHandlers{
			Telegram:             NewTelegramService(svc),
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
