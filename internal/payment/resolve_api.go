package payment

import (
	"context"

	"espx/internal/config"
)

func OpenAPIOrDial(ctx context.Context, cfg *config.Config) (PaymentAPI, func(), error) {
	noop := func() {}
	if cfg == nil || string(cfg.PaymentInternalToken) == "" {
		return nil, noop, nil
	}
	token := string(cfg.PaymentInternalToken)
	mod, err := OpenModule(ctx, cfg)
	if err != nil {
		return nil, noop, err
	}
	if mod == nil {
		return nil, noop, nil
	}
	return mod.API(token), mod.Close, nil
}
