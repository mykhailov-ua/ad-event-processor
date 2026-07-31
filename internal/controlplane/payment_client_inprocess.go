package controlplane

import (
	"context"

	"espx/internal/config"
	"espx/internal/payment"
)

func NewPaymentClientInProcess(api payment.PaymentAPI, token string) *PaymentClient {
	return NewPaymentClientFromAPI(api, token)
}

func openPaymentClient(ctx context.Context, cfg *config.Config, opts ServeOptions) (*PaymentClient, func(), error) {
	noop := func() {}
	if opts.Payment != nil {
		return opts.Payment, noop, nil
	}
	if cfg != nil && !cfg.PaymentGRPCEnabled {
		mod, err := payment.OpenModule(ctx, cfg)
		if err != nil {
			return nil, noop, err
		}
		if mod == nil {
			return nil, noop, nil
		}
		token := string(cfg.PaymentInternalToken)
		return NewPaymentClientFromAPI(mod.API(token), token), mod.Close, nil
	}
	client, err := NewPaymentClient(cfg)
	closeFn := noop
	if client != nil {
		closeFn = func() { _ = client.Close() }
	}
	return client, closeFn, err
}
