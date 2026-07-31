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
	noop := func() {}
	if opts.Billing != nil {
		return opts.Billing, noop, nil
	}
	if cfg != nil && !cfg.BillingGRPCEnabled {
		mod, err := billing.OpenModule(ctx, cfg)
		if err != nil {
			return nil, noop, err
		}
		if mod == nil {
			return nil, noop, nil
		}
		token := string(cfg.BillingInternalToken)
		return NewBillingClientFromAPI(mod.API(token), token), mod.Close, nil
	}
	client, err := NewBillingClient(cfg)
	closeFn := noop
	if client != nil {
		closeFn = func() { _ = client.Close() }
	}
	return client, closeFn, err
}
