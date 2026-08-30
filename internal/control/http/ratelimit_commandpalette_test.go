package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ad-event-processor/internal/controlplane/authz"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestCommandPalette_rateLimit_holdout(t *testing.T) {
	t.Parallel()
	lim := NewCommandPaletteSearchLimiter()
	key := "holdout-user"
	start := time.Now()
	require.Equal(t, CommandPaletteSearchBurst, lim.AllowNAt(key, start, CommandPaletteSearchBurst))
	for i := 0; i < 25; i++ {
		at := start.Add(time.Duration((i+1)*2) * time.Second)
		require.True(t, lim.AllowAt(key, at), "paced request %d", i+1)
	}
	require.False(t, lim.AllowAt(key, start.Add(50*time.Second)), "31st request in window should be rate limited")
}

func TestLimitCommandPaletteSearch_returns429(t *testing.T) {
	t.Parallel()
	lim := NewCommandPaletteSearchLimiter()
	userID := uuid.MustParse("00000000-0000-4000-8000-000000000099")
	handler := LimitCommandPaletteSearch(lim, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	send := func() int {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/command-palette/search", http.NoBody)
		req = req.WithContext(authz.WithAuthenticatedUser(req.Context(), authz.AuthenticatedUser{
			UserID: userID,
		}))
		rec := httptest.NewRecorder()
		handler(rec, req)
		return rec.Code
	}
	for i := 0; i < CommandPaletteSearchBurst; i++ {
		require.Equal(t, http.StatusOK, send(), "burst request %d", i+1)
	}
	require.Equal(t, http.StatusTooManyRequests, send())
}
