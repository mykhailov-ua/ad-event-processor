package adminapi

import (
	"context"

	"espx/internal/billing"
	"espx/internal/payment"

	"github.com/google/uuid"
	"time"
)

type CreateCampaignInput struct {
	CustomerID       uuid.UUID
	BrandID          *uuid.UUID
	Name             string
	BudgetLimitMicro int64
	PacingMode       string
	DailyBudgetMicro int64
	Timezone         string
	FreqLimit        int32
	FreqWindow       int32
	TargetCountries  []string
	StartAt          *time.Time
	EndAt            *time.Time
	DaypartHours     []int16
	IdempotencyKey   string
}

type CampaignAdmin interface {
	EnforceSelfServeCreateLimits(ctx context.Context, customerID uuid.UUID, budgetMicro int64) error
	GenerateIdempotencyHash(customerID uuid.UUID, payload []byte) (string, error)
	CreateCampaign(ctx context.Context, spec CreateCampaignInput) (uuid.UUID, error)
	PauseCampaign(ctx context.Context, campaignID uuid.UUID, reason string) error
	ResumeCampaign(ctx context.Context, campaignID uuid.UUID, reason string) error
}

type PaymentIntents = payment.PaymentAPI

type APIKeyResult struct {
	ID         string
	Name       string
	RawKey     string
	ExpiresAt  string
	HasExpires bool
}

type APIKeyCreator interface {
	CreateAPIKey(ctx context.Context, accessToken, name string) (APIKeyResult, error)
}

type InvoiceLister = billing.BillingAPI
