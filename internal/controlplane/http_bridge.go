package controlplane

import (
	"context"

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

func wireSessionHTTPHandlers(freshness func(context.Context) reports.DataFreshnessDTO) *platformadmin.SessionHTTPHandlers {
	return &platformadmin.SessionHTTPHandlers{
		Freshness:     freshness,
		BuildNav:      buildSessionNav,
		ResolveUser:   resolveSessionUser,
		NormalizeRole: authz.NormalizeRole,
	}
}
