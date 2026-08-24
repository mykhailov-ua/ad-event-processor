package costsync

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestFetchRichAdsCosts_Httptest(t *testing.T) {
	customerID := uuid.New()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/api/reports/", r.URL.Path)
		require.Equal(t, "2026-08-24", r.URL.Query().Get("from"))
		require.Equal(t, "2026-08-24", r.URL.Query().Get("to"))
		require.Equal(t, "campaign_id", r.URL.Query().Get("segment"))
		require.Equal(t, "json", r.URL.Query().Get("output"))
		require.Equal(t, "rich-key", r.URL.Query().Get("api_key"))

		_, _ = w.Write([]byte(`{
			"response": {
				"result": [
					{"campaign_id": 77101, "spend": 8.25, "impressions": 1200}
				]
			}
		}`))
	}))
	defer srv.Close()

	client := &http.Client{Transport: roundTripRewriteHost(srv.URL, nil)}
	cred := Credential{
		CustomerID: customerID,
		APIKey:     "rich-key",
	}
	date := time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC)
	lines, err := fetchRichAdsCosts(context.Background(), client, srv.URL, cred, date)
	require.NoError(t, err)
	require.Len(t, lines, 1)
	require.Equal(t, "richads", lines[0].Network)
	require.Equal(t, int64(8_250_000), lines[0].AmountMicro)
	require.Equal(t, "77101", lines[0].PlacementID)
}
