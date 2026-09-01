package controlplane

// http_bridge: cold-path adapters between Handler/Service and domain HTTP (auth, session nav, ops reader).
import (
	"context"
	"log/slog"

	"ad-event-processor/internal/database"
	"ad-event-processor/internal/opsadmin"

	"ad-event-processor/internal/config"
	ctrlhttp "ad-event-processor/internal/control/http"
	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/internal/identity"
	"ad-event-processor/internal/ledger"
	"ad-event-processor/internal/notify"
	"ad-event-processor/internal/payment"
	"ad-event-processor/internal/platformadmin"
	"ad-event-processor/internal/reports"
	"ad-event-processor/internal/shardadmin"
	"ad-event-processor/internal/telegram"
	"ad-event-processor/pkg/legal"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func openPaymentClient(_ context.Context, _ *config.Config, opts ServeOptions) (*payment.APIClient, func(), error) {
	if opts.Payment != nil {
		return opts.Payment, func() {}, nil
	}
	return nil, func() {}, nil
}

func openBillingClient(_ context.Context, _ *config.Config, opts ServeOptions) (*ledger.BillingClient, func(), error) {
	if opts.Billing != nil {
		return opts.Billing, func() {}, nil
	}
	return nil, func() {}, nil
}

func openNotifierClient(_ context.Context, _ *config.Config, opts ServeOptions) (*notify.Client, func(), error) {
	if opts.Notifier != nil {
		return opts.Notifier, func() {}, nil
	}
	return nil, func() {}, nil
}

type platformAuthAdapter struct {
	client *identity.AuthClient
}

func (a platformAuthAdapter) Register(ctx context.Context, adminAPIKey, email, password, role, customerID string) error {
	if a.client == nil {
		return identity.ErrAuthUnavailable
	}
	_, err := a.client.Register(ctx, adminAPIKey, email, password, role, customerID)
	return err
}

// NewAuthHandler: /api/v1/auth/* login and token refresh; picks healthy control Redis shard for session state.
func NewAuthHandler(
	authClient *identity.AuthClient,
	tokenMaker identity.Maker,
	redisShards []redis.UniversalClient,
	cfg *config.Config,
	authMiddleware *AuthMiddleware,
) *ctrlhttp.AuthHandler {
	return ctrlhttp.NewAuthHandler(authClient, tokenMaker, redisShards, cfg, authMiddleware, authMiddleware, shardadmin.PickHealthyControlShard)
}

func apiKeyPrincipalID(apiKey string) uuid.UUID {
	return ctrlhttp.APIKeyPrincipalID(apiKey)
}

func newIPRateLimiter(rps float64, burst int) *ctrlhttp.IPRateLimiter {
	return ctrlhttp.NewIPRateLimiter(rps, burst)
}

func newAPIKeyRateLimiter(rps float64, burst int) *ctrlhttp.APIKeyRateLimiter {
	return ctrlhttp.NewAPIKeyRateLimiter(rps, burst)
}

func newCustomerRateLimiterWith(rps float64, burst int) *ctrlhttp.CustomerRateLimiter {
	return ctrlhttp.NewCustomerRateLimiterWith(rps, burst)
}

func isAdminSPAPath(path string) bool {
	return ctrlhttp.IsAdminSPAPath(path)
}

var _ telegram.Host = (*Service)(nil)

func NewTelegramService(svc *Service) *telegram.Service {
	return telegram.NewService(svc)
}

// buildSessionNav: RBAC snapshot from request context; no per-nav-item PG round-trip.
func buildSessionNav(ctx context.Context) []platformadmin.SessionNavItemDTO {
	catalogRows := reports.FilterReportCatalog(ctx, reports.ReportCatalogEntries)
	items := []platformadmin.SessionNavItemDTO{
		{Href: "/campaigns", Label: "Campaigns", IconKey: "campaigns"},
		{Href: "/dashboards/buyer", Label: "Dashboard", IconKey: "dashboard"},
		{Href: "/reports", Label: "Reports", IconKey: "reports"},
	}
	snap, ok := authz.SnapshotFromContext(ctx)
	if ok && snap.Has("customers:read") && snap.Mask == authz.MaskFull {
		items = append(items, platformadmin.SessionNavItemDTO{Href: "/customers", Label: "Customers", IconKey: "customers"})
	}
	if ok && snap.Has("audit:read") {
		items = append(items, platformadmin.SessionNavItemDTO{Href: "/ops", Label: "Operations", IconKey: "ops"})
	}
	for _, row := range catalogRows {
		if row.Category != "fraud" {
			continue
		}
		items = append(items, platformadmin.SessionNavItemDTO{
			Href:    "/reports/" + row.Key,
			Label:   row.Title,
			IconKey: "report",
		})
	}
	return items
}

func resolveBootstrapAuthUser(ctx context.Context) (platformadmin.BootstrapUserDTO, bool) {
	u, ok := GetUser(ctx)
	if !ok {
		return platformadmin.BootstrapUserDTO{}, false
	}
	role := authz.NormalizeRole(u.Role)
	return platformadmin.BootstrapUserDTO{
		ID:          u.UserID.String(),
		Role:        role,
		CustomerID:  u.CustomerID.String(),
		Permissions: ctrlhttp.GetPermissionsForRole(u.Role),
	}, true
}

func resolveEulaBootstrap(ctx context.Context, svc *Service) (platformadmin.EulaBootstrapDTO, error) {
	out := platformadmin.EulaBootstrapDTO{}
	if svc == nil {
		return out, nil
	}
	_, accepted, err := svc.GetEulaStatus(ctx)
	if err != nil {
		return out, err
	}
	out.EulaAccepted = accepted
	out.EulaRequired = !accepted
	out.EulaVersion = legal.Version
	return out, nil
}

func resolveSessionUser(ctx context.Context) (platformadmin.SessionUser, bool) {
	u, ok := GetUser(ctx)
	if !ok {
		return platformadmin.SessionUser{}, false
	}
	out := platformadmin.SessionUser{Role: u.Role}
	if u.HasBoundCustomer() {
		out.DefaultCustomerID = u.CustomerID.String()
	}
	return out, true
}

func wireSessionHTTPHandlers(
	svc *Service,
	freshness func(context.Context) reports.DataFreshnessDTO,
) *platformadmin.SessionHTTPHandlers {
	return &platformadmin.SessionHTTPHandlers{
		Freshness:       freshness,
		BuildNav:        buildSessionNav,
		ResolveUser:     resolveSessionUser,
		ResolveAuthUser: resolveBootstrapAuthUser,
		EulaSnapshot: func(ctx context.Context) (platformadmin.EulaBootstrapDTO, error) {
			return resolveEulaBootstrap(ctx, svc)
		},
		NormalizeRole: authz.NormalizeRole,
	}
}

var _ opsadmin.OpsMetricScraperHost = (*Service)(nil)

func (s *Service) startOpsMetricScraper(ctx context.Context, scrapeURL string) {
	opsadmin.StartMetricScraper(s, ctx, scrapeURL)
}

// NewManagementOpsReader: ops stack health fan-out (PG pool, Redis shards, optional ClickHouseQuery readonly).
func NewManagementOpsReader(svc *Service) opsadmin.ManagementOpsReader {
	if svc == nil {
		return nil
	}
	var clickhouseQuery *database.ClickHouseQuery
	if svc.ClickHouseQuery() != nil {
		clickhouseQuery = svc.ClickHouseQuery()
	}
	return opsadmin.NewReader(opsadmin.ReaderDeps{
		Pool:        svc.GetPool(),
		RedisShards: svc.redisShards,
		Config:      svc.cfg,
		GetShardHealth: func(ctx context.Context) (opsadmin.ShardHealthReport, error) {
			report, err := svc.GetShardHealth(ctx)
			return report, err
		},
		ListReconRuns: svc.ListReconRuns,
		BuildStackHealthSnapshot: func(ctx context.Context) (opsadmin.StackHealthSnapshot, error) {
			return svc.BuildStackHealthSnapshot(ctx)
		},
		ClickHouseQuery: clickhouseQuery,
	})
}

func newOpsReader(svc *Service) opsadmin.ManagementOpsReader {
	return NewManagementOpsReader(svc)
}

// StartFilterRejectRollupWorker: background CH->PG rollup; requires PG pool and clickhouseQuery (readonly).
func (s *Service) StartFilterRejectRollupWorker(ctx context.Context, scrapeURL string) {
	if s == nil || s.GetPool() == nil || s.clickhouseQuery == nil {
		slog.Warn("filter reject rollup worker not started: postgres or clickhouse unavailable")
		return
	}
	w := opsadmin.NewFilterRejectRollupWorker(s.GetPool(), s.clickhouseQuery, scrapeURL)
	w.SetEdgeFetcher(func(ctx context.Context) (map[string]uint64, error) {
		panel, err := opsadmin.FetchEdgeMetrics(ctx)
		if err != nil {
			return nil, err
		}
		return panel.Blocked, nil
	})
	s.StartBackgroundWorker(func() {
		w.Start(ctx)
	})
}
