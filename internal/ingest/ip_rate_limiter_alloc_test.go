package ingest

import (
	"context"
	"testing"
	"time"

	"ad-event-processor/internal/domain"

	"github.com/stretchr/testify/require"
)

func TestIPRateLimiter_Check_allocBudget(t *testing.T) {
	redisClient := &mockRedisClient{}
	l := NewIPRateLimiter(redisClient, 100, 10*time.Minute)
	evt := &domain.Event{IP: "192.168.1.1"}
	ctx := context.Background()
	for range 32 {
		require.NoError(t, l.Check(ctx, evt))
	}
	allocs := testing.AllocsPerRun(100, func() {
		_ = l.Check(ctx, evt)
	})
	// Not on /track hot path: stack-local redis wire boxes key/cmd args (race-free vs shared cache).
	const allocBudget = 4.0
	if allocs > allocBudget {
		t.Fatalf("IPRateLimiter.Check allocs/op = %v, want <= %v", allocs, allocBudget)
	}
}
