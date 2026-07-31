package controlplane

import (
	"espx/internal/config"
	"espx/internal/payment"
)

func NewPaymentClientInProcess(api payment.PaymentAPI, token string) *PaymentClient {
	return NewPaymentClientFromAPI(api, token)
}

func openPaymentClient(cfg *config.Config, opts ServeOptions) (*PaymentClient, error) {
	if opts.Payment != nil {
		return opts.Payment, nil
	}
	return NewPaymentClient(cfg)
}
