package domain

import (
	"context"
	"time"
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
	IntentID       string
	Status         string
	CheckoutURL    string
	ProviderRef    string
	DepositAddress string
	DepositNetwork string
	DepositQRSVG   string
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
