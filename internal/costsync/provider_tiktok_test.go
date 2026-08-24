package costsync

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

func TestFetchTikTokCosts_Httptest(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok)
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(filename), "testdata", "tiktok_report_integrated.json"))
	require.NoError(t, err)

	customerID := uuid.New()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/open_api/v1.3/report/integrated/get/", r.URL.Path)
		require.Equal(t, "adv-42", r.URL.Query().Get("advertiser_id"))
		require.Equal(t, "tok-abc", r.Header.Get("Access-Token"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(raw)
	}))
	defer srv.Close()

	client := &http.Client{Transport: roundTripRewriteHost(srv.URL, nil)}
	cred := Credential{
		CustomerID:  customerID,
		AccountID:   "adv-42",
		AccessToken: "tok-abc",
	}
	date := time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC)
	lines, err := fetchTikTokCosts(context.Background(), client, srv.URL+"/open_api/v1.3", cred, date)
	require.NoError(t, err)
	require.Len(t, lines, 1)
	require.Equal(t, "tiktok", lines[0].Network)
	require.Equal(t, int64(12_500_000), lines[0].AmountMicro)
	require.Equal(t, "camp-99", lines[0].PlacementID)
}
