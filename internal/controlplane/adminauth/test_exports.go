package adminauth

import (
	"context"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/identity"

	"github.com/redis/go-redis/v9"
)

func CheckTokenRevocationForTest(ctx context.Context, redisShards []redis.UniversalClient, cfg *config.Config, payload *identity.Payload) (bool, error) {
	m := &Middleware{cfg: cfg}
	return m.checkTokenRevocation(ctx, redisShards, payload)
}
