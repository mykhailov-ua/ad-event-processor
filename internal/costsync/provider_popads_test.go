package costsync

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestFetchPopAdsCosts_Httptest(t *testing.T) {
	customerID := uuid.New()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/api/report_advertiser", r.URL.Path)
		require.Equal(t, "application/json", r.Header.Get("Accept"))

		raw, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		form, err := url.ParseQuery(string(raw))
		require.NoError(t, err)
		require.Equal(t, "pop-key", form.Get("key"))
		require.Equal(t, "UTC", form.Get("zone"))
		require.Equal(t, "campaign", form.Get("groups"))
		require.Equal(t, "2026-08-24 00:00", form.Get("start"))
		require.Equal(t, "2026-08-24 23:59", form.Get("end"))

		_ = json.NewEncoder(w).Encode(map[string]any{
			"rows": []map[string]any{
				{"campaign_id": "55221", "cost": 3.75},
			},
		})
	}))
	defer srv.Close()

	client := &http.Client{Transport: roundTripRewriteHost(srv.URL, nil)}
	cred := Credential{CustomerID: customerID, APIKey: "pop-key"}
	date := time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC)
	lines, err := fetchPopAdsCosts(context.Background(), client, srv.URL, cred, date)
	require.NoError(t, err)
	require.Len(t, lines, 1)
	require.Equal(t, "popads", lines[0].Network)
	require.Equal(t, int64(3_750_000), lines[0].AmountMicro)
	require.Equal(t, "55221", lines[0].PlacementID)
}
