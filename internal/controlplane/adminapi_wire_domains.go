package controlplane

import (
	"context"
	"net/http"
	"time"

	"ad-event-processor/internal/automation"
	"ad-event-processor/internal/billingadmin"
	"ad-event-processor/internal/brand"
	"ad-event-processor/internal/campaign"
	"ad-event-processor/internal/campaign/integration"
	"ad-event-processor/internal/campaign/selfserve"
	"ad-event-processor/internal/commandpalette"
	ctrlhttp "ad-event-processor/internal/control/http"
	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/internal/dashboardadmin"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/edge"
	"ad-event-processor/internal/flow"
	"ad-event-processor/internal/fraudadmin"
	"ad-event-processor/internal/licensingadmin"
	"ad-event-processor/internal/marginguard"
	"ad-event-processor/internal/openrtb"
	"ad-event-processor/internal/opsadmin"
	"ad-event-processor/internal/platformadmin"
	"ad-event-processor/internal/reportjob"
	"ad-event-processor/internal/reports"
	"ad-event-processor/internal/reports/views"
	"ad-event-processor/internal/rtbadmin"
	"ad-event-processor/internal/supply"
	"ad-event-processor/internal/telegram"
	"ad-event-processor/internal/trafficoptimizer"
	"ad-event-processor/pkg/platformconfig"

	"github.com/google/uuid"
)

// wireAdminDomainRoutes: phase-2 RouteRegistry fill; handlers own route paths and coldpath body limits.
func (h *Handler) wireAdminDomainRoutes(reg *RouteRegistry, e adminWireEnv) {
	pool := e.pool
	svc := e.svc
	limit := e.limit
	perm := e.perm
	permAny := e.permAny
	selfServePerm := e.selfServePerm
	writeErr := e.writeErr
	authCustomer := e.authCustomer
	authCampaign := e.authCampaign
	reportJobs := e.reportJobs
	encKey := e.encKey
	costWorker := e.costWorker
	platformWorker := e.platformWorker
	selfServePaymentProvider := e.selfServePaymentProvider
	selfServeCryptoSubProvider := e.selfServeCryptoSubProvider
	fraudPresets := e.fraudPresets
	// ReportsHTTP: CH readonly stats/forecasts; PG views store; license feature gate on premium catalog rows.
	reg.ReportsHTTP = &reports.ReportsHTTPHandlers{
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
		DenyScopedAPIKeyReport:    selfserve.DenyScopedAPIKeyOperatorReport,
	}
	// ReportJobHTTP: async CH/PG export jobs; schedule validation is server-side only.
	reg.ReportJobHTTP = &reportjob.HTTPHandlers{
		Runner:                  reportJobs,
		Pool:                    pool,
		ApplyRateLimit:          limit,
		RequirePermission:       perm,
		RequireAnyPermission:    permAny,
		AuthorizeCustomerAccess: authCustomer,
		ValidateReportSchedule:  views.ValidateReportScheduleForActor,
		WriteServiceError:       writeErr,
	}
	reg.DashboardsHTTP = &dashboardadmin.HTTPHandlers{
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
	}
	reg.ViewsHTTP = &reports.ViewsHTTPHandlers{
		Store:                   reports.NewViewsStore(pool),
		ApplyRateLimit:          limit,
		RequirePermission:       perm,
		RequireAnyPermission:    permAny,
		AuthorizeCustomerAccess: authCustomer,
		WriteServiceError:       writeErr,
	}
	// SelfServeHTTP: /api/v1/selfserve/*; API-key auth via selfServePerm; customer scope enforced in resolver.
	reg.SelfServeHTTP = &selfserve.SelfServeHTTPHandlers{
		Campaigns:                  svc,
		Templates:                  selfserve.NewSelfServeTemplatesAdapter(svc),
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
	}
	reg.PostbackHTTP = &campaign.PostbackHTTPHandlers{
		Pool:              pool,
		EncryptionKey:     encKey,
		ApplyRateLimit:    limit,
		RequirePermission: perm,
	}
	reg.CostSyncHTTP = &billingadmin.CostSyncHTTPHandlers{
		Pool:              pool,
		EncryptionKey:     encKey,
		Worker:            costWorker,
		ApplyRateLimit:    limit,
		RequirePermission: perm,
	}
	reg.PlatformCampaignHTTP = &platformadmin.PlatformCampaignHTTPHandlers{
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
	}
	reg.MarginGuardHTTP = &marginguard.HTTPHandlers{
		Service:           marginGuardServiceAdapter{svc: svc},
		ApplyRateLimit:    limit,
		RequirePermission: perm,
	}
	reg.SmartAlertsHTTP = &SmartAlertsHTTPHandlers{
		Service:           svc.SmartAlertsStore(),
		ApplyRateLimit:    limit,
		RequirePermission: perm,
		ResolveActorID: func(ctx context.Context) uuid.UUID {
			u, ok := GetUser(ctx)
			if !ok {
				return uuid.Nil
			}
			return u.UserID
		},
	}
	reg.AutomationHTTP = &automation.HTTPHandlers{
		Rules:             svc.AutomationRules(),
		ApplyRateLimit:    limit,
		RequirePermission: perm,
	}
	reg.TrafficOptimizerHTTP = &trafficoptimizer.HTTPHandlers{
		Rules:             svc.TrafficOptimizerRules(),
		ApplyRateLimit:    limit,
		RequirePermission: perm,
	}
	reg.CommandPaletteHTTP = &commandpalette.HTTPHandlers{
		Search:                         svc.CommandPaletteService(),
		Recents:                        svc.CommandPaletteService().Recents,
		LicenseFeatureAllowed:          licenseFeatureAllowed,
		ApplyCommandPaletteSearchLimit: h.limitCommandPaletteSearch,
		ApplyRateLimit:                 limit,
		RequireAnyPermission:           permAny,
		ResolveCustomerID:              h.resolveCampaignsCustomerID,
		MaxQueryLen:                    h.cfg.Management.CommandPaletteMaxQLen,
		AuditLogEnabled:                h.cfg.Management.CommandPaletteAuditLog,
	}
	reg.DomainHealthHTTP = &platformadmin.DomainHealthHTTPHandlers{
		Service:           svc,
		ApplyRateLimit:    limit,
		RequirePermission: perm,
		TLSAskToken:       string(h.cfg.Management.CaddyTLSAskToken),
		TLSAskAllowLocal:  h.cfg.Management.CaddyTLSAskAllowLocal,
	}
	reg.FlowHTTP = &flow.HTTPHandlers{
		Service:           svc,
		ApplyRateLimit:    limit,
		RequirePermission: perm,
	}
	reg.IntegrationSchemaHTTP = &integration.IntegrationSchemaHTTPHandlers{
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
	}
	reg.TeamHTTP = &platformadmin.TeamHTTPHandlers{
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
	}
	reg.PublisherHTTP = &dashboardadmin.PublisherHTTPHandlers{
		Publisher:            svc,
		ApplyRateLimit:       limit,
		RequireAnyPermission: permAny,
		ActorUserID: func(r *http.Request) (uuid.UUID, bool) {
			u, ok := GetUser(r.Context())
			return u.UserID, ok
		},
		WriteServiceError: writeErr,
	}
	reg.BrandHTTP = &brand.HTTPHandlers{
		Admin:                   brand.NewAdminAdapter(svc.BrandStore()),
		ApplyRateLimit:          limit,
		RequirePermission:       perm,
		AuthorizeCustomerAccess: authCustomer,
		WriteServiceError:       writeErr,
	}
	reg.SupplyHTTP = &supply.HTTPHandlers{
		Admin:                supply.NewAdminAdapter(supplyAdminHost{svc: svc}),
		ApplyRateLimit:       limit,
		RequirePermission:    perm,
		RequireAnyPermission: permAny,
		WriteServiceError:    writeErr,
	}
	reg.RtbFloorsHTTP = &rtbadmin.FloorsHTTPHandlers{
		Service:           svc,
		ApplyRateLimit:    limit,
		RequirePermission: perm,
		WriteServiceError: writeErr,
	}
	reg.RtbHTTP = &rtbadmin.HTTPHandlers{
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
	}
	// CampaignsHTTP: PG mutations via CampaignRuntime; ClickHouseQuery read-only for event stats.
	reg.CampaignsHTTP = &campaign.CampaignsHTTPHandlers{
		Campaigns:                 svc.CampaignRuntime(),
		CampaignFraud:             fraudadmin.CampaignFraudAPI{Host: svc, MapErr: mapFraudadminErr},
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
	}
	reg.FraudHTTP = &fraudadmin.HTTPHandlers{
		Labels:                  fraudadmin.LabelsAPI{Host: svc},
		Decisions:               fraudadmin.DecisionsAPI{Host: svc},
		Integrations:            fraudadmin.IntegrationsAPI{Pool: svc.GetPool(), MapErr: mapFraudadminErr},
		Overrides:               fraudadmin.OverridesAPI{Host: svc, MapErr: mapFraudadminErr},
		Presets:                 fraudPresets,
		ApplyRateLimit:          limit,
		AllowFraudDecision:      h.allowFraudDecision,
		RequirePermission:       perm,
		RequireAnyPermission:    permAny,
		ResolveCustomerID:       h.resolveCampaignsCustomerID,
		AuthorizeCampaignAccess: authCampaign,
		WriteServiceError:       writeErr,
	}
	reg.CustomersHTTP = &platformadmin.CustomersHTTPHandlers{
		Customers:               svc,
		CostCenter:              svc,
		ApplyRateLimit:          limit,
		RequirePermission:       perm,
		AuthorizeCustomerAccess: authCustomer,
		WriteServiceError:       writeErr,
	}
	// SupportHTTP: auth-only (no perm string); support bundle streams PG metadata + local log dir.
	reg.SupportHTTP = &platformadmin.SupportHTTPHandlers{
		Feedback: svc,
		SupportBundle: supportBundleWriter{
			pool:   pool,
			logDir: h.cfg.Logger.Dir,
		},
		ApplyRateLimit:    limit,
		RequireAuth:       h.adminRequireAuth(),
		WriteServiceError: writeErr,
	}
	reg.MetaHTTP = &platformadmin.MetaHTTPHandlers{
		ApplyRateLimit: limit,
		Enrich:         platformadmin.NewMetaEnricher(h.svc),
		WriteError:     writeErr,
	}
	// SessionHTTP: SPA bootstrap; CH ingestion lag probe when clickhouseQuery configured (readonly).
	reg.SessionHTTP = func() *platformadmin.SessionHTTPHandlers {
		sh := wireSessionHTTPHandlers(h.svc, func(ctx context.Context) reports.DataFreshnessDTO {
			if h.svc != nil && h.svc.clickhouseQuery != nil {
				lag, _ := h.svc.clickHouseIngestionLag(ctx)
				return portfolioFreshness(time.Now().UTC(), true, lag)
			}
			return reports.DataFreshnessDTO{Consistency: "eventual"}
		})
		sh.ApplyRateLimit = limit
		sh.RequireAuth = h.adminRequireAuth()
		return sh
	}()
	reg.EulaHTTP = &licensingadmin.EulaHTTPHandlers{
		Service:           svc,
		ApplyRateLimit:    limit,
		RequirePermission: perm,
		WriteServiceError: writeErr,
	}
	reg.PlatformHTTP = &platformadmin.HTTPHandlers{
		Service:           svc,
		AuthClient:        platformAuthAdapter{client: h.authClient},
		Cfg:               h.cfg,
		ApplyRateLimit:    limit,
		RequirePermission: perm,
		WriteServiceError: writeErr,
	}
	// PublicHTTP: unauthenticated activation; license-apply IP limit; PolicyRefresh reloads authz roles YAML.
	reg.PublicHTTP = &platformadmin.PublicHTTPHandlers{
		Activation: platformadmin.NewPublicActivation(svc),
		AuthClient: h.authClient,
		ApplyRateLimit: func(next http.HandlerFunc) http.HandlerFunc {
			return limit(ctrlhttp.LimitLicenseApply(h.licenseApplyLimiter, next))
		},
		WriteServiceError: writeErr,
		PolicyRefresh:     h.authMiddleware,
	}
	reg.StubHTTP = &StubHTTPHandlers{
		ApplyRateLimit:    limit,
		RequirePermission: perm,
	}
	reg.TelegramHTTP = &telegram.HTTPHandlers{
		Telegram:             NewTelegramService(svc),
		ApplyRateLimit:       limit,
		RequireAnyPermission: permAny,
		WriteServiceError:    writeErr,
	}
}
