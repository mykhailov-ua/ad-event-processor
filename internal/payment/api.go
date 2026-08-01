package payment

import (
	"context"
	"time"

	"espx/internal/config"

	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
)

type Handler struct {
	service *Service
	cfg     *config.Config
}

func NewHandler(service *Service, cfg *config.Config) *Handler {
	return &Handler{
		service: service,
		cfg:     cfg,
	}
}

type PaymentIntent struct {
	ID             string
	CustomerID     string
	AmountMicro    int64
	Currency       string
	Status         string
	Provider       string
	ProviderRef    string
	IdempotencyKey string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type CreatePaymentIntentResult struct {
	IntentID    string
	Status      string
	CheckoutURL string
	ProviderRef string
}

type ListPaymentIntentsResult struct {
	Intents []PaymentIntent
	Total   int64
}

type Dispute struct {
	IntentID          string
	CustomerID        string
	AmountMicro       int64
	Currency          string
	ProviderDisputeID string
	UpdatedAt         time.Time
}

type ListDisputesResult struct {
	Disputes []Dispute
	Total    int64
}

type PaymentAPI interface {
	CreatePaymentIntent(ctx context.Context, customerID string, amountMicro int64, currency, idempotencyKey string, meta map[string]string) (*CreatePaymentIntentResult, error)
	ListPaymentIntents(ctx context.Context, customerID string, limit, offset int32) (ListPaymentIntentsResult, error)
	ListDisputes(ctx context.Context, customerID string, limit, offset int32) (ListDisputesResult, error)
	ReplayWebhook(ctx context.Context, provider, providerEventID string) (string, error)
}

type paymentAPI struct {
	h     *Handler
	token string
}

func (m *Module) API(token string) PaymentAPI {
	if m == nil || m.Handler == nil {
		return nil
	}
	return &paymentAPI{h: m.Handler, token: token}
}

func (a *paymentAPI) incoming(ctx context.Context) context.Context {
	if a.token == "" {
		return ctx
	}
	return metadata.NewIncomingContext(ctx, metadata.Pairs("x-internal-token", a.token))
}

func (a *paymentAPI) CreatePaymentIntent(ctx context.Context, customerID string, amountMicro int64, currency, idempotencyKey string, meta map[string]string) (*CreatePaymentIntentResult, error) {
	customerUUID, err := uuid.Parse(customerID)
	if err != nil {
		return nil, err
	}
	result, err := a.h.createPaymentIntent(a.incoming(ctx), customerUUID, amountMicro, currency, idempotencyKey, meta)
	if err != nil {
		return nil, err
	}
	return &CreatePaymentIntentResult{
		IntentID:    result.Intent.ID,
		Status:      result.Intent.Status,
		CheckoutURL: result.CheckoutURL,
		ProviderRef: result.Intent.ProviderRef,
	}, nil
}

func (a *paymentAPI) ListPaymentIntents(ctx context.Context, customerID string, limit, offset int32) (ListPaymentIntentsResult, error) {
	customerUUID, err := uuid.Parse(customerID)
	if err != nil {
		return ListPaymentIntentsResult{}, err
	}
	intents, total, err := a.h.listPaymentIntents(a.incoming(ctx), customerUUID, limit, offset)
	if err != nil {
		return ListPaymentIntentsResult{}, err
	}
	return ListPaymentIntentsResult{Intents: intents, Total: total}, nil
}

func (a *paymentAPI) ListDisputes(ctx context.Context, customerID string, limit, offset int32) (ListDisputesResult, error) {
	var customerUUID *uuid.UUID
	if customerID != "" {
		parsed, err := uuid.Parse(customerID)
		if err != nil {
			return ListDisputesResult{}, err
		}
		customerUUID = &parsed
	}
	disputes, total, err := a.h.listDisputes(a.incoming(ctx), customerUUID, limit, offset)
	if err != nil {
		return ListDisputesResult{}, err
	}
	return ListDisputesResult{Disputes: disputes, Total: total}, nil
}

func (a *paymentAPI) ReplayWebhook(ctx context.Context, provider, providerEventID string) (string, error) {
	return a.h.replayWebhook(a.incoming(ctx), provider, providerEventID)
}
