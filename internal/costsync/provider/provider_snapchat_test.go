package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestFetchSnapchatCosts_Httptest(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok)
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(filename), "testdata", "snapchat_adaccount_stats.json"))
	require.NoError(t, err)

	customerID := uuid.New()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v1/adaccounts/acct-77/stats", r.URL.Path)
		require.Equal(t, "campaign", r.URL.Query().Get("breakdown"))
		require.Equal(t, "DAY", r.URL.Query().Get("granularity"))
		require.Equal(t, "spend", r.URL.Query().Get("fields"))
		require.Equal(t, "Bearer snap-tok", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(raw)
	}))
	defer srv.Close()

	client := &http.Client{Transport: roundTripRewriteHost(srv.URL, nil)}
	cred := Credential{
		CustomerID:  customerID,
		AccountID:   "acct-77",
		AccessToken: "snap-tok",
	}
	date := time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC)
	lines, err := fetchSnapchatCosts(context.Background(), client, srv.URL+"/v1", cred, date)
	require.NoError(t, err)
	require.Len(t, lines, 1)
	require.Equal(t, "snapchat", lines[0].Network)
	require.Equal(t, int64(12_500_000), lines[0].AmountMicro)
	require.Equal(t, "camp-snap-42", lines[0].PlacementID)
}
