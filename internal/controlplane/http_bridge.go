package controlplane

import (
	"context"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/controlplane/authz"
	ctrlhttp "ad-event-processor/internal/control/http"
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

type AuthHandler = ctrlhttp.AuthHandler

type UserDTO = ctrlhttp.UserDTO

type LoginRequest = ctrlhttp.LoginRequest

type RegisterRequest = ctrlhttp.RegisterRequest

type (
	IPRateLimiter       = ctrlhttp.IPRateLimiter
	APIKeyRateLimiter   = ctrlhttp.APIKeyRateLimiter
	CustomerRateLimiter = ctrlhttp.CustomerRateLimiter
)

var (
	SetTrustedProxies         = ctrlhttp.SetTrustedProxies
	NewCORSMiddleware         = ctrlhttp.NewCORSMiddleware
	NewCSRFMiddleware         = ctrlhttp.NewCSRFMiddleware
	SecurityHeadersMiddleware = ctrlhttp.SecurityHeadersMiddleware
	GenerateSecureToken       = ctrlhttp.GenerateSecureToken
	NormalizeRole             = ctrlhttp.NormalizeRole
	GetPermissionsForRole     = ctrlhttp.GetPermissionsForRole
	HasPermission             = ctrlhttp.HasPermission
)

const (
	PermCustomersRead        = ctrlhttp.PermCustomersRead
	PermCustomersWrite       = ctrlhttp.PermCustomersWrite
	PermCampaignsRead        = ctrlhttp.PermCampaignsRead
	PermCampaignsWrite       = ctrlhttp.PermCampaignsWrite
	PermCampaignsReadMasked  = ctrlhttp.PermCampaignsReadMasked
	PermCampaignsWriteMasked = ctrlhttp.PermCampaignsWriteMasked
	PermCampaignsPause       = ctrlhttp.PermCampaignsPause
	PermBillingRead          = ctrlhttp.PermBillingRead
	PermBillingWrite         = ctrlhttp.PermBillingWrite
	PermBrandsRead           = ctrlhttp.PermBrandsRead
	PermBrandsWrite          = ctrlhttp.PermBrandsWrite
	PermSettingsRead         = ctrlhttp.PermSettingsRead
	PermSettingsWrite        = ctrlhttp.PermSettingsWrite
	PermBlacklistRead        = ctrlhttp.PermBlacklistRead
	PermBlacklistWrite       = ctrlhttp.PermBlacklistWrite
	PermAuditRead            = ctrlhttp.PermAuditRead
	PermUsersWrite           = ctrlhttp.PermUsersWrite
	PermShardsRead           = ctrlhttp.PermShardsRead
	PermShardsWrite          = ctrlhttp.PermShardsWrite
	PermOpsWrite             = ctrlhttp.PermOpsWrite
	PermRtbRead              = ctrlhttp.PermRtbRead
	PermRtbWrite             = ctrlhttp.PermRtbWrite
	PermSupplyReadScoped     = ctrlhttp.PermSupplyReadScoped
)

const (
	RoleAdmin      = ctrlhttp.RoleAdmin
	RoleManager    = ctrlhttp.RoleManager
	RoleUser       = ctrlhttp.RoleUser
	RoleBuyer      = ctrlhttp.RoleBuyer
	RoleSupport    = ctrlhttp.RoleSupport
	RoleTeamLead   = ctrlhttp.RoleTeamLead
	RoleMediaBuyer = ctrlhttp.RoleMediaBuyer
	RolePublisher  = ctrlhttp.RolePublisher
)

func NewAuthHandler(
	authClient *identity.AuthClient,
	tokenMaker identity.Maker,
	redisShards []redis.UniversalClient,
	cfg *config.Config,
	authMiddleware *AuthMiddleware,
) *AuthHandler {
	return ctrlhttp.NewAuthHandler(authClient, tokenMaker, redisShards, cfg, authMiddleware, authMiddleware, shardadmin.PickHealthyControlShard)
}

func apiKeyPrincipalID(apiKey string) uuid.UUID {
	return ctrlhttp.APIKeyPrincipalID(apiKey)
}

func newIPRateLimiter(rps float64, burst int) *IPRateLimiter {
	return ctrlhttp.NewIPRateLimiter(rps, burst)
}

func newAPIKeyRateLimiter(rps float64, burst int) *APIKeyRateLimiter {
	return ctrlhttp.NewAPIKeyRateLimiter(rps, burst)
}

func newCustomerRateLimiter() *CustomerRateLimiter {
	return ctrlhttp.NewCustomerRateLimiter()
}

func newCustomerRateLimiterWith(rps float64, burst int) *CustomerRateLimiter {
	return ctrlhttp.NewCustomerRateLimiterWith(rps, burst)
}

func newFraudDecisionLimiter() *CustomerRateLimiter {
	return ctrlhttp.NewFraudDecisionLimiter()
}

func newFraudPreviewLimiter() *CustomerRateLimiter {
	return ctrlhttp.NewFraudPreviewLimiter()
}

func isAdminSPAPath(path string) bool {
	return ctrlhttp.IsAdminSPAPath(path)
}

type (
	TelegramHTTPHandlers = telegram.HTTPHandlers
	TelegramService      = telegram.TelegramService

	ValidateResult       = telegram.ValidateResult
	ClickMintResult      = telegram.ClickMintResult
	BotDTO               = telegram.BotDTO
	DeeplinkDTO          = telegram.DeeplinkDTO
	PostbackDTO          = telegram.PostbackDTO
	TelegramReportFilter = telegram.ReportFilter
)

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

func wireSessionHTTPHandlers(freshness func(context.Context) DataFreshnessDTO) *SessionHTTPHandlers {
	return &SessionHTTPHandlers{
		Freshness:     freshness,
		BuildNav:      buildSessionNav,
		ResolveUser:   resolveSessionUser,
		NormalizeRole: authz.NormalizeRole,
	}
}
