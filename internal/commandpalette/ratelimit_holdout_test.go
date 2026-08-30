package commandpalette

import (
	"testing"
	"time"

	"golang.org/x/time/rate"

	"github.com/stretchr/testify/require"
)

func TestCommandPalette_rateLimit_holdout(t *testing.T) {
	t.Parallel()
	lim := rate.NewLimiter(rate.Limit(30.0/60.0), 5)
	start := time.Now()
	require.Equal(t, 5, allowNAt(lim, start, 5))
	for i := 0; i < 25; i++ {
		at := start.Add(time.Duration((i+1)*2) * time.Second)
		require.True(t, lim.AllowN(at, 1), "paced request %d", i+1)
	}
	require.False(t, lim.AllowN(start.Add(50*time.Second), 1), "31st request in window should be rate limited")
}

func allowNAt(lim *rate.Limiter, at time.Time, n int) int {
	allowed := 0
	for i := 0; i < n; i++ {
		if !lim.AllowN(at, 1) {
			break
		}
		allowed++
	}
	return allowed
}
