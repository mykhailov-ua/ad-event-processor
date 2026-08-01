package controlplane

import (
	"context"
	"errors"
	"fmt"
	"time"

	"espx/internal/billing"
	"espx/internal/config"
	"espx/internal/identity"
	"espx/internal/notifier"
	"espx/internal/payment"
)

var errAuthUnavailable = errors.New("auth service not configured")

type AuthClient struct {
	api identity.AuthAPI
}

func NewAuthClientFromAPI(api identity.AuthAPI) *AuthClient {
	if api == nil {
		return nil
	}
	return &AuthClient{api: api}
}

func (c *AuthClient) VerifyAPIKey(ctx context.Context, apiKey string) (identity.AuthUser, error) {
	if c == nil || c.api == nil {
		return identity.AuthUser{}, errAuthUnavailable
	}
	return c.api.VerifyAPIKey(ctx, apiKey)
}

func (c *AuthClient) CreateAPIKey(ctx context.Context, bearerToken, name string) (identity.CreateAPIKeyResult, error) {
	if c == nil || c.api == nil {
		return identity.CreateAPIKeyResult{}, errAuthUnavailable
	}
	return c.api.CreateAPIKey(ctx, bearerToken, name)
}

func (c *AuthClient) Login(ctx context.Context, email, password string, durationHours int32) (identity.LoginResult, error) {
	if c == nil || c.api == nil {
		return identity.LoginResult{}, errAuthUnavailable
	}
	return c.api.Login(ctx, email, password, durationHours)
}

func (c *AuthClient) Register(ctx context.Context, adminAPIKey, email, password, role, customerID string) (identity.RegisterResult, error) {
	if c == nil || c.api == nil {
		return identity.RegisterResult{}, errAuthUnavailable
	}
	return c.api.Register(ctx, adminAPIKey, email, password, role, customerID)
}

func (c *AuthClient) RefreshToken(ctx context.Context, refreshToken string) (identity.RefreshResult, error) {
	if c == nil || c.api == nil {
		return identity.RefreshResult{}, errAuthUnavailable
	}
	return c.api.RefreshToken(ctx, refreshToken)
}

func (c *AuthClient) RevokeToken(ctx context.Context, refreshToken string) error {
	if c == nil || c.api == nil {
		return errAuthUnavailable
	}
	return c.api.RevokeToken(ctx, refreshToken)
}

func TryAuthClient(ctx context.Context, cfg *config.Config) (*AuthClient, func(), error) {
	api, closeFn, err := identity.OpenAPI(ctx, cfg)
	if err != nil || api == nil {
		return nil, closeFn, err
	}
	return NewAuthClientFromAPI(api), closeFn, nil
}

var _ billing.BillingAPI = (*BillingClient)(nil)

type BillingClient struct {
	api   billing.BillingAPI
	token string
}

func NewBillingClientFromAPI(api billing.BillingAPI, token string) *BillingClient {
	if api == nil || token == "" {
		return nil
	}
	return &BillingClient{api: api, token: token}
}

func NewBillingClientInProcess(api billing.BillingAPI, token string) *BillingClient {
	return NewBillingClientFromAPI(api, token)
}

func openBillingClient(ctx context.Context, cfg *config.Config, opts ServeOptions) (*BillingClient, func(), error) {
	if opts.Billing != nil {
		return opts.Billing, func() {}, nil
	}
	token := ""
	if cfg != nil {
		token = string(cfg.BillingInternalToken)
	}
	api, closeFn, err := billing.OpenAPI(ctx, cfg)
	if err != nil || api == nil {
		return nil, closeFn, err
	}
	return NewBillingClientFromAPI(api, token), closeFn, nil
}

func (client *BillingClient) Close() error {
	return nil
}

func (client *BillingClient) GenerateInvoice(ctx context.Context, customerID string, billingMonth time.Time) (*billing.Invoice, error) {
	if client == nil || client.api == nil {
		return nil, fmt.Errorf("billing client not configured")
	}
	return client.api.GenerateInvoice(ctx, customerID, billingMonth)
}

func (client *BillingClient) GetInvoice(ctx context.Context, invoiceID string) (*billing.Invoice, error) {
	if client == nil || client.api == nil {
		return nil, fmt.Errorf("billing client not configured")
	}
	return client.api.GetInvoice(ctx, invoiceID)
}

func (client *BillingClient) ListInvoices(ctx context.Context, customerID string, limit, offset int32) (billing.ListInvoicesResult, error) {
	if client == nil || client.api == nil {
		return billing.ListInvoicesResult{}, fmt.Errorf("billing client not configured")
	}
	return client.api.ListInvoices(ctx, customerID, limit, offset)
}

var _ payment.PaymentAPI = (*PaymentClient)(nil)

type PaymentClient struct {
	api   payment.PaymentAPI
	token string
}

func NewPaymentClientFromAPI(api payment.PaymentAPI, token string) *PaymentClient {
	if api == nil || token == "" {
		return nil
	}
	return &PaymentClient{api: api, token: token}
}

func NewPaymentClientInProcess(api payment.PaymentAPI, token string) *PaymentClient {
	return NewPaymentClientFromAPI(api, token)
}

func openPaymentClient(ctx context.Context, cfg *config.Config, opts ServeOptions) (*PaymentClient, func(), error) {
	if opts.Payment != nil {
		return opts.Payment, func() {}, nil
	}
	token := ""
	if cfg != nil {
		token = string(cfg.PaymentInternalToken)
	}
	api, closeFn, err := payment.OpenAPI(ctx, cfg)
	if err != nil || api == nil {
		return nil, closeFn, err
	}
	return NewPaymentClientFromAPI(api, token), closeFn, nil
}

func (c *PaymentClient) Close() error {
	return nil
}

func (c *PaymentClient) CreatePaymentIntent(ctx context.Context, customerID string, amountMicro int64, currency, idempotencyKey string, meta map[string]string) (*payment.CreatePaymentIntentResult, error) {
	if c == nil || c.api == nil {
		return nil, fmt.Errorf("payment client not configured")
	}
	return c.api.CreatePaymentIntent(ctx, customerID, amountMicro, currency, idempotencyKey, meta)
}

func (c *PaymentClient) ListPaymentIntents(ctx context.Context, customerID string, limit, offset int32) (payment.ListPaymentIntentsResult, error) {
	if c == nil || c.api == nil {
		return payment.ListPaymentIntentsResult{}, fmt.Errorf("payment client not configured")
	}
	return c.api.ListPaymentIntents(ctx, customerID, limit, offset)
}

func (c *PaymentClient) ListDisputes(ctx context.Context, customerID string, limit, offset int32) (payment.ListDisputesResult, error) {
	if c == nil || c.api == nil {
		return payment.ListDisputesResult{}, fmt.Errorf("payment client not configured")
	}
	return c.api.ListDisputes(ctx, customerID, limit, offset)
}

func (c *PaymentClient) ReplayWebhook(ctx context.Context, provider, providerEventID string) (string, error) {
	if c == nil || c.api == nil {
		return "", fmt.Errorf("payment client not configured")
	}
	return c.api.ReplayWebhook(ctx, provider, providerEventID)
}

type NotifierClient struct {
	closeFn func()
	api     notifier.NotifierAPI
}

func NewNotifierClientFromAPI(api notifier.NotifierAPI) *NotifierClient {
	if api == nil {
		return nil
	}
	return &NotifierClient{api: api}
}

func NewNotifierClient(ctx context.Context, cfg *config.Config) (*NotifierClient, error) {
	if cfg == nil || !cfg.NotifierDialEnabled() {
		return nil, nil
	}
	api, closeFn, err := notifier.OpenAPI(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if api == nil {
		return nil, nil
	}
	return &NotifierClient{api: api, closeFn: closeFn}, nil
}

func NewNotifierClientInProcess(api notifier.NotifierAPI) *NotifierClient {
	return NewNotifierClientFromAPI(api)
}

func TryNotifierClient(ctx context.Context, cfg *config.Config) (*NotifierClient, func(), error) {
	if cfg == nil || !cfg.NotifierDialEnabled() {
		return nil, func() {}, nil
	}
	api, closeFn, err := notifier.OpenAPI(ctx, cfg)
	if err != nil {
		return nil, func() {}, err
	}
	if api == nil {
		return nil, func() {}, nil
	}
	return NewNotifierClientFromAPI(api), closeFn, nil
}

func openNotifierClient(ctx context.Context, cfg *config.Config, opts ServeOptions) (*NotifierClient, func(), error) {
	if opts.Notifier != nil {
		return opts.Notifier, func() {}, nil
	}
	return TryNotifierClient(ctx, cfg)
}

func (client *NotifierClient) API() notifier.NotifierAPI {
	if client == nil {
		return nil
	}
	return client.api
}

func (client *NotifierClient) Close() error {
	if client == nil || client.closeFn == nil {
		return nil
	}
	client.closeFn()
	client.closeFn = nil
	return nil
}
