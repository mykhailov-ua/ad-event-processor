package payment

import (
	"context"
	"time"

	"espx/internal/payment/pb"

	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
)

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

func PaymentIntentFromPB(in *pb.PaymentIntent) *PaymentIntent {
	if in == nil {
		return nil
	}
	out := &PaymentIntent{
		ID:             in.Id,
		CustomerID:     in.CustomerId,
		AmountMicro:    in.AmountMicro,
		Currency:       in.Currency,
		Status:         in.Status.String(),
		Provider:       in.Provider,
		ProviderRef:    in.ProviderRef,
		IdempotencyKey: in.IdempotencyKey,
	}
	if in.CreatedAt != nil {
		out.CreatedAt = in.CreatedAt.AsTime().UTC()
	}
	if in.UpdatedAt != nil {
		out.UpdatedAt = in.UpdatedAt.AsTime().UTC()
	}
	return out
}

func CreatePaymentIntentResultFromPB(resp *pb.CreatePaymentIntentResponse) *CreatePaymentIntentResult {
	if resp == nil {
		return nil
	}
	return &CreatePaymentIntentResult{
		IntentID:    resp.IntentId,
		Status:      resp.Status.String(),
		CheckoutURL: resp.CheckoutUrl,
		ProviderRef: resp.ProviderRef,
	}
}

func ListPaymentIntentsResultFromPB(resp *pb.ListPaymentIntentsResponse) ListPaymentIntentsResult {
	if resp == nil {
		return ListPaymentIntentsResult{}
	}
	out := ListPaymentIntentsResult{Total: resp.Total}
	if len(resp.Intents) == 0 {
		return out
	}
	out.Intents = make([]PaymentIntent, 0, len(resp.Intents))
	for _, in := range resp.Intents {
		if parsed := PaymentIntentFromPB(in); parsed != nil {
			out.Intents = append(out.Intents, *parsed)
		}
	}
	return out
}

func DisputeFromPB(d *pb.DisputeRecord) *Dispute {
	if d == nil {
		return nil
	}
	out := &Dispute{
		IntentID:          d.IntentId,
		CustomerID:        d.CustomerId,
		AmountMicro:       d.AmountMicro,
		Currency:          d.Currency,
		ProviderDisputeID: d.ProviderDisputeId,
	}
	if d.UpdatedAt != nil {
		out.UpdatedAt = d.UpdatedAt.AsTime().UTC()
	}
	return out
}

func ListDisputesResultFromPB(resp *pb.ListDisputesResponse) ListDisputesResult {
	if resp == nil {
		return ListDisputesResult{}
	}
	out := ListDisputesResult{Total: resp.Total}
	if len(resp.Disputes) == 0 {
		return out
	}
	out.Disputes = make([]Dispute, 0, len(resp.Disputes))
	for _, d := range resp.Disputes {
		if parsed := DisputeFromPB(d); parsed != nil {
			out.Disputes = append(out.Disputes, *parsed)
		}
	}
	return out
}
