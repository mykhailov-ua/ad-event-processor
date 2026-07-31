package controlplane

import (
	"context"

	"espx/internal/billing"
	"espx/internal/config"
)

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
	api, closeFn, err := billing.OpenAPIOrDial(ctx, cfg)
	if err != nil || api == nil {
		return nil, closeFn, err
	}
	return NewBillingClientFromAPI(api, token), closeFn, nil
}
