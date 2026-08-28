package telegram

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRateLimiter_PerChat(t *testing.T) {
	t.Parallel()
	l := NewRateLimiter()
	ctx := context.Background()
	start := time.Now()
	for range 3 {
		require.NoError(t, l.Wait(ctx, 42, false))
	}
	elapsed := time.Since(start)
	require.GreaterOrEqual(t, elapsed, 2*time.Second)
}

func TestRateLimiter_GlobalBurst(t *testing.T) {
	t.Parallel()
	l := NewRateLimiter()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	const n = 35
	done := make(chan error, n)
	for i := range n {
		chatID := int64(1000 + i)
		go func(id int64) {
			done <- l.Wait(ctx, id, false)
		}(chatID)
	}
	for range n {
		require.NoError(t, <-done)
	}
}
