package controlplane

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTelegramRateLimiter_PerChat(t *testing.T) {
	t.Parallel()
	l := NewTelegramRateLimiter()
	ctx := context.Background()
	start := time.Now()
	for i := 0; i < 3; i++ {
		require.NoError(t, l.Wait(ctx, 42, false))
	}
	elapsed := time.Since(start)
	require.GreaterOrEqual(t, elapsed, 2*time.Second)
}

func TestTelegramRateLimiter_GlobalBurst(t *testing.T) {
	t.Parallel()
	l := NewTelegramRateLimiter()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	const n = 35
	done := make(chan error, n)
	for i := 0; i < n; i++ {
		chatID := int64(1000 + i)
		go func(id int64) {
			done <- l.Wait(ctx, id, false)
		}(chatID)
	}
	for i := 0; i < n; i++ {
		require.NoError(t, <-done)
	}
}
