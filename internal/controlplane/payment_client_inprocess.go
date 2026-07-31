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
	if opts.Payment != nil {
		return opts.Payment, func() {}, nil
	}
	token := ""
	if cfg != nil {
		token = string(cfg.PaymentInternalToken)
	}
	api, closeFn, err := payment.OpenAPIOrDial(ctx, cfg)
	if err != nil || api == nil {
		return nil, closeFn, err
	}
	return NewPaymentClientFromAPI(api, token), closeFn, nil
}
