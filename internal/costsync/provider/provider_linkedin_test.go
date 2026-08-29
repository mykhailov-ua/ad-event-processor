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

func TestFetchLinkedInCosts_Httptest(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok)
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(filename), "testdata", "linkedin_ad_analytics.json"))
	require.NoError(t, err)

	customerID := uuid.New()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/rest/adAnalytics", r.URL.Path)
		require.Equal(t, "analytics", r.URL.Query().Get("q"))
		require.Equal(t, "CAMPAIGN", r.URL.Query().Get("pivot"))
		require.Equal(t, "DAILY", r.URL.Query().Get("timeGranularity"))
		require.Contains(t, r.URL.Query().Get("accounts"), "urn:li:sponsoredAccount:502840441")
		require.Equal(t, "Bearer li-tok", r.Header.Get("Authorization"))
		require.Equal(t, "202508", r.Header.Get("Linkedin-Version"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(raw)
	}))
	defer srv.Close()

	client := &http.Client{Transport: roundTripRewriteHost(srv.URL, nil)}
	cred := Credential{
		CustomerID:  customerID,
		AccountID:   "502840441",
		AccessToken: "li-tok",
	}
	date := time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC)
	lines, err := fetchLinkedInCosts(context.Background(), client, srv.URL+"/rest", cred, date)
	require.NoError(t, err)
	require.Len(t, lines, 1)
	require.Equal(t, "linkedin", lines[0].Network)
	require.Equal(t, int64(18_750_000), lines[0].AmountMicro)
	require.Equal(t, "88042", lines[0].PlacementID)
}
