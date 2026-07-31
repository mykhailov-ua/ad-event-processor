package control

import (
	"context"

	"espx/internal/billing"
	"espx/internal/config"
	"espx/internal/controlplane"
	"espx/internal/identity"
	"espx/internal/notifier"
	"espx/internal/payment"
)

func buildServeOptions(ctx context.Context, cfg *config.Config, opts Options) (controlplane.ServeOptions, []func(), error) {
	out := controlplane.ServeOptions{}
	var cleanups []func()

	if opts.Auth {
		mod, err := identity.OpenModule(ctx, cfg)
		if err != nil {
			return out, cleanups, err
		}
		if mod != nil {
			out.Auth = controlplane.NewAuthClientFromAPI(mod.API())
			cleanups = append(cleanups, mod.Close)
		}
	}
	if opts.Billing {
		mod, err := billing.OpenModule(ctx, cfg)
		if err != nil {
			return out, cleanups, err
		}
		if mod != nil {
			out.BillingModule = mod
			out.Billing = controlplane.NewBillingClientFromAPI(mod.API(string(cfg.BillingInternalToken)), string(cfg.BillingInternalToken))
			cleanups = append(cleanups, mod.Close)
		}
	}
	if opts.Payment {
		mod, err := payment.OpenModule(ctx, cfg)
		if err != nil {
			return out, cleanups, err
		}
		if mod != nil {
			out.PaymentModule = mod
			out.Payment = controlplane.NewPaymentClientFromAPI(mod.API(string(cfg.PaymentInternalToken)), string(cfg.PaymentInternalToken))
			cleanups = append(cleanups, mod.Close)
		}
	}
	if opts.Notifier {
		mod, err := notifier.OpenModule(ctx, cfg)
		if err != nil {
			return out, cleanups, err
		}
		if mod != nil {
			out.NotifierModule = mod
			out.Notifier = controlplane.NewNotifierClientFromAPI(mod.API())
			cleanups = append(cleanups, mod.Close)
		}
	}
	return out, cleanups, nil
}
