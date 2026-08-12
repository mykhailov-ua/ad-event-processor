package controlplane

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/bidshard/ad-event-processor/internal/identity"
	"github.com/bidshard/ad-event-processor/internal/notify"
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

var _ domain.BillingAPI = (*BillingClient)(nil)

type BillingClient struct {
	api   domain.BillingAPI
	token string
}

func NewBillingClientFromAPI(api domain.BillingAPI, token string) *BillingClient {
	if api == nil || token == "" {
		return nil
	}
	return &BillingClient{api: api, token: token}
}

func NewBillingClientInProcess(api domain.BillingAPI, token string) *BillingClient {
	return NewBillingClientFromAPI(api, token)
}

func openBillingClient(_ context.Context, _ *config.Config, opts ServeOptions) (*BillingClient, func(), error) {
	if opts.Billing != nil {
		return opts.Billing, func() {}, nil
	}
	return nil, func() {}, nil
}

func (client *BillingClient) Close() error {
	return nil
}

func (client *BillingClient) GenerateInvoice(ctx context.Context, customerID string, billingMonth time.Time) (*domain.Invoice, error) {
	if client == nil || client.api == nil {
		return nil, fmt.Errorf("billing client not configured")
	}
	return client.api.GenerateInvoice(ctx, customerID, billingMonth)
}

func (client *BillingClient) GetInvoice(ctx context.Context, invoiceID string) (*domain.Invoice, error) {
	if client == nil || client.api == nil {
		return nil, fmt.Errorf("billing client not configured")
	}
	return client.api.GetInvoice(ctx, invoiceID)
}

func (client *BillingClient) ListInvoices(ctx context.Context, customerID string, limit, offset int32) (domain.ListInvoicesResult, error) {
	if client == nil || client.api == nil {
		return domain.ListInvoicesResult{}, fmt.Errorf("billing client not configured")
	}
	return client.api.ListInvoices(ctx, customerID, limit, offset)
}

var _ domain.PaymentAPI = (*PaymentClient)(nil)

type PaymentClient struct {
	api   domain.PaymentAPI
	token string
}

func NewPaymentClientFromAPI(api domain.PaymentAPI, token string) *PaymentClient {
	if api == nil || token == "" {
		return nil
	}
	return &PaymentClient{api: api, token: token}
}

func NewPaymentClientInProcess(api domain.PaymentAPI, token string) *PaymentClient {
	return NewPaymentClientFromAPI(api, token)
}

func openPaymentClient(_ context.Context, _ *config.Config, opts ServeOptions) (*PaymentClient, func(), error) {
	if opts.Payment != nil {
		return opts.Payment, func() {}, nil
	}
	return nil, func() {}, nil
}

func (c *PaymentClient) Close() error {
	return nil
}

func (c *PaymentClient) CreatePaymentIntent(ctx context.Context, customerID string, amountMicro int64, currency, idempotencyKey string, meta map[string]string) (*domain.CreatePaymentIntentResult, error) {
	if c == nil || c.api == nil {
		return nil, fmt.Errorf("payment client not configured")
	}
	return c.api.CreatePaymentIntent(ctx, customerID, amountMicro, currency, idempotencyKey, meta)
}

func (c *PaymentClient) ListPaymentIntents(ctx context.Context, customerID string, limit, offset int32) (domain.ListPaymentIntentsResult, error) {
	if c == nil || c.api == nil {
		return domain.ListPaymentIntentsResult{}, fmt.Errorf("payment client not configured")
	}
	return c.api.ListPaymentIntents(ctx, customerID, limit, offset)
}

func (c *PaymentClient) ListDisputes(ctx context.Context, customerID string, limit, offset int32) (domain.ListDisputesResult, error) {
	if c == nil || c.api == nil {
		return domain.ListDisputesResult{}, fmt.Errorf("payment client not configured")
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
	api     notify.NotifierAPI
}

func NewNotifierClientFromAPI(api notify.NotifierAPI) *NotifierClient {
	if api == nil {
		return nil
	}
	return &NotifierClient{api: api}
}

func NewNotifierClientInProcess(api notify.NotifierAPI) *NotifierClient {
	return NewNotifierClientFromAPI(api)
}

func openNotifierClient(_ context.Context, _ *config.Config, opts ServeOptions) (*NotifierClient, func(), error) {
	if opts.Notifier != nil {
		return opts.Notifier, func() {}, nil
	}
	return nil, func() {}, nil
}

func (client *NotifierClient) API() notify.NotifierAPI {
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
