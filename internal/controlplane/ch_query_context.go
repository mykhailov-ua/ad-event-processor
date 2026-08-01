package controlplane

import (
	"context"
	"time"
)

const coldPathCHQueryTimeout = 10 * time.Second

func chQueryContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, coldPathCHQueryTimeout)
}
