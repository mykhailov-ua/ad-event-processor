package billing

import (
	"context"

	"espx/internal/config"
)

func OpenAPIOrDial(ctx context.Context, cfg *config.Config) (BillingAPI, func(), error) {
	noop := func() {}
	if cfg == nil || string(cfg.BillingInternalToken) == "" {
		return nil, noop, nil
	}
	token := string(cfg.BillingInternalToken)
	mod, err := OpenModule(ctx, cfg)
	if err != nil {
		return nil, noop, err
	}
	if mod == nil {
		return nil, noop, nil
	}
	return mod.API(token), mod.Close, nil
}
