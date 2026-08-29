package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestFetchEvadavCosts_Httptest(t *testing.T) {
	customerID := uuid.New()
	var body map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/advertiser/stats/campaign", r.URL.Path)
		require.Equal(t, "ev-key", r.Header.Get("X-Api-Key"))
		raw, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(raw, &body))
		require.Equal(t, "24.08.2026", body["day"])
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data": map[string]any{
				"stat": []map[string]any{
					{"campaignId": 77, "cost": 3.25},
				},
			},
		})
	}))
	defer srv.Close()

	client := &http.Client{Transport: roundTripRewriteHost(srv.URL, nil)}
	cred := Credential{
		CustomerID: customerID,
		APIKey:     "ev-key",
	}
	date := time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC)
	lines, err := fetchEvadavCosts(context.Background(), client, srv.URL, cred, date)
	require.NoError(t, err)
	require.Len(t, lines, 1)
	require.Equal(t, "evadav", lines[0].Network)
	require.Equal(t, int64(3_250_000), lines[0].AmountMicro)
	require.Equal(t, "77", lines[0].PlacementID)
}
