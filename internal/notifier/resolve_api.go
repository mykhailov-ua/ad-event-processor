package notifier

import (
	"context"

	"espx/internal/config"
)

func OpenAPIOrDial(ctx context.Context, cfg *config.Config) (NotifierAPI, func(), error) {
	noop := func() {}
	if cfg == nil {
		return nil, noop, nil
	}
	mod, err := OpenModule(ctx, cfg)
	if err != nil {
		return nil, noop, err
	}
	if mod == nil {
		return nil, noop, nil
	}
	return mod.API(), mod.Close, nil
}
