package checkout

import (
	"context"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
)

type DisputeOps interface {
	ListDisputes(ctx context.Context, customerID *uuid.UUID, limit, offset int32) ([]DisputeListItem, int64, error)
	ReplayWebhook(ctx context.Context, provider, providerEventID string) (string, error)
}

type Handler struct {
	service  *Service
	disputes DisputeOps
	cfg      *config.Config
}

func NewHandler(service *Service, disputes DisputeOps, cfg *config.Config) *Handler {
	return &Handler{
		service:  service,
		disputes: disputes,
		cfg:      cfg,
	}
}

type PaymentAPI = domain.PaymentAPI

type paymentAPI struct {
	h     *Handler
	token string
}

func NewPaymentAPI(h *Handler, token string) PaymentAPI {
	if h == nil {
		return nil
	}
	return &paymentAPI{h: h, token: token}
}

func (a *paymentAPI) incoming(ctx context.Context) context.Context {
	if a.token == "" {
		return ctx
	}
	return metadata.NewIncomingContext(ctx, metadata.Pairs("x-internal-token", a.token))
}

func (a *paymentAPI) CreatePaymentIntent(ctx context.Context, customerID string, amountMicro int64, currency, idempotencyKey string, meta map[string]string) (*domain.CreatePaymentIntentResult, error) {
	customerUUID, err := uuid.Parse(customerID)
	if err != nil {
		return nil, err
	}
	result, err := a.h.createPaymentIntent(a.incoming(ctx), customerUUID, amountMicro, currency, idempotencyKey, meta)
	if err != nil {
		return nil, err
	}
	return &domain.CreatePaymentIntentResult{
		IntentID:       result.Intent.ID,
		Status:         result.Intent.Status,
		CheckoutURL:    result.CheckoutURL,
		ProviderRef:    result.Intent.ProviderRef,
		DepositAddress: result.DepositAddress,
		DepositNetwork: result.DepositNetwork,
		DepositQRSVG:   result.DepositQRSVG,
	}, nil
}

func (a *paymentAPI) ListPaymentIntents(ctx context.Context, customerID string, limit, offset int32) (domain.ListPaymentIntentsResult, error) {
	customerUUID, err := uuid.Parse(customerID)
	if err != nil {
		return domain.ListPaymentIntentsResult{}, err
	}
	intents, total, err := a.h.listPaymentIntents(a.incoming(ctx), customerUUID, limit, offset)
	if err != nil {
		return domain.ListPaymentIntentsResult{}, err
	}
	return domain.ListPaymentIntentsResult{Intents: intents, Total: total}, nil
}

func (a *paymentAPI) ListDisputes(ctx context.Context, customerID string, limit, offset int32) (domain.ListDisputesResult, error) {
	var customerUUID *uuid.UUID
	if customerID != "" {
		parsed, err := uuid.Parse(customerID)
		if err != nil {
			return domain.ListDisputesResult{}, err
		}
		customerUUID = &parsed
	}
	disputes, total, err := a.h.listDisputes(a.incoming(ctx), customerUUID, limit, offset)
	if err != nil {
		return domain.ListDisputesResult{}, err
	}
	return domain.ListDisputesResult{Disputes: disputes, Total: total}, nil
}

func (a *paymentAPI) ReplayWebhook(ctx context.Context, provider, providerEventID string) (string, error) {
	return a.h.replayWebhook(a.incoming(ctx), provider, providerEventID)
}
