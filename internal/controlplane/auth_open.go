package controlplane

import (
	"context"

	"espx/internal/config"
	"espx/internal/identity"
)

func TryAuthClient(ctx context.Context, cfg *config.Config) (*AuthClient, func(), error) {
	api, closeFn, err := identity.OpenAPIOrDial(ctx, cfg)
	if err != nil || api == nil {
		return nil, closeFn, err
	}
	return NewAuthClientFromAPI(api), closeFn, nil
}
