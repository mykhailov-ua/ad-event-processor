package controlplane

import (
	"espx/internal/billing"
	"espx/internal/config"
)

func NewBillingClientInProcess(api billing.BillingAPI, token string) *BillingClient {
	return NewBillingClientFromAPI(api, token)
}

func openBillingClient(cfg *config.Config, opts ServeOptions) (*BillingClient, error) {
	if opts.Billing != nil {
		return opts.Billing, nil
	}
	return NewBillingClient(cfg)
}
