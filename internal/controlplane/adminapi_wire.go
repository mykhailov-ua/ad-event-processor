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

	"ad-event-processor/internal/billingadmin"
	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/internal/costsync"
	"ad-event-processor/internal/doctor"
	"ad-event-processor/internal/edge"
	"ad-event-processor/internal/fraudadmin"
	"ad-event-processor/internal/licensing"
	"ad-event-processor/internal/licensingadmin"
	"ad-event-processor/internal/opsadmin"
	"ad-event-processor/internal/payment"
	"ad-event-processor/internal/platformsync"
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
	if r.mw == nil {
		return fmt.Errorf("policy store not configured")
	}
	return r.mw.ReloadRolesYAML()
}

func (r rolesReloader) RolesPath() string { return authz.DefaultRolesPath() }

// BuildAdminAPIRegistry: cold-path RouteRegistry for RegisterRoutes (register.go); no HTTP listen here.
// Returns empty registry when pool or svc nil so RegisterRoutes is a no-op until serve.go wires PG.
func (h *Handler) BuildAdminAPIRegistry(pool *pgxpool.Pool, redisShards []redis.UniversalClient) RouteRegistry {
	if h == nil || h.svc == nil || pool == nil {
		return RouteRegistry{}
	}

	// Shared middleware closures: limit = IP rate limit + PostgresGate high slot; perm* = AuthMiddleware RBAC.
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
	// CompositeReadService: PG ledger truth; ClickHouseQuery is readonly reporting conn (CH_READONLY_DSN).
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
	fraudPresets := fraudadmin.PresetsAPI{Host: svc, Pool: svc.GetPool(), MapErr: mapFraudadminErr}
	reportJobs := h.svc.ReportJobRunner()
	if reportJobs == nil {
		reportJobs = h.svc.InitReportJobRunner(reportExportDirFromWire())
	}

	doctorHTTP := &doctor.DoctorHTTPHandlers{
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
	}

	// Phase-1 registry: billing, ops, doctor, licensing; webhooks and export omit standard perm stack.
	reg := RouteRegistry{
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
		// CryptoBillingWebhook: provider POST callbacks; signature verified in handler, not operator JWT.
		CryptoBillingWebhook: &billingadmin.CryptoWebhookHandlers{
			Processor:           svc,
			CryptoWebhookSecret: string(h.cfg.CryptoWebhookSecret),
			BTCPayWebhookSecret: string(h.cfg.BTCPayWebhookSecret),
			CryptomusAPIKey:     string(h.cfg.CryptomusAPIKey),
		},
		// DoctorHTTP ProbeDeps: synchronous cold probes (Redis shards, PG pool, license, edge XDP snapshot).
		DoctorHTTP: doctorHTTP,
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
			DoctorSnapshot:          doctorHTTP.BuildSnapshot,
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
	}
	// Phase-2: campaigns, reports, fraud, platform, self-serve; same adminWireEnv middleware and scope hooks.
	h.wireAdminDomainRoutes(&reg, adminWireEnv{
		pool:                       pool,
		svc:                        svc,
		encKey:                     encKey,
		costWorker:                 costWorker,
		platformWorker:             platformWorker,
		selfServePaymentProvider:   selfServePaymentProvider,
		selfServeCryptoSubProvider: selfServeCryptoSubProvider,
		fraudPresets:               fraudPresets,
		reportJobs:                 reportJobs,
		limit:                      limit,
		perm:                       perm,
		permAny:                    permAny,
		selfServePerm:              selfServePerm,
		writeErr:                   writeErr,
		authCustomer:               authCustomer,
		authCampaign:               authCampaign,
	})
	return reg
}
