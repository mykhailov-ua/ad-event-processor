package controlplane

import (
	"context"

	"espx/internal/domain"
	"espx/internal/notifier"
)

type InProcessPaymentModule interface {
	SetSettlementAPI(api domain.PaymentSettlement)
	SetNotifierAPI(api notifier.NotifierAPI)
	StartWorkers(ctx context.Context)
}

type InProcessBillingModule interface {
	ConfigureNotifier(api notifier.NotifierAPI)
	StartWorkers(ctx context.Context)
}

type InProcessNotifierModule interface {
	StartWorkers(ctx context.Context)
}

type ServeOptions struct {
	Auth           *AuthClient
	Billing        *BillingClient
	Payment        *PaymentClient
	Notifier       *NotifierClient
	BillingModule  InProcessBillingModule
	PaymentModule  InProcessPaymentModule
	NotifierModule InProcessNotifierModule
	RtbBidShadeSim RtbBidShadeSimulator
}

func (o ServeOptions) Monolith() bool {
	return o.Auth != nil && o.Billing != nil && o.Payment != nil && o.Notifier != nil
}
