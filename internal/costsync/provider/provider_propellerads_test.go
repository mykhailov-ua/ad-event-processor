package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestFetchPropellerAdsCosts_Httptest(t *testing.T) {
	customerID := uuid.New()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v5/adv/statistics", r.URL.Path)
		require.Equal(t, "Bearer pa-token", r.Header.Get("Authorization"))

		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.Equal(t, "2026-08-24", body["day_from"])
		require.Equal(t, "2026-08-24", body["day_to"])

		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": []map[string]any{
				{"campaign_id": 9001, "spent": 8.25},
			},
		})
	}))
	defer srv.Close()

	client := &http.Client{Transport: roundTripRewriteHost(srv.URL, nil)}
	cred := Credential{
		CustomerID: customerID,
		APIKey:     "pa-token",
	}
	date := time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC)
	lines, err := fetchPropellerAdsCosts(context.Background(), client, srv.URL+"/v5", cred, date)
	require.NoError(t, err)
	require.Len(t, lines, 1)
	require.Equal(t, "propellerads", lines[0].Network)
	require.Equal(t, int64(8_250_000), lines[0].AmountMicro)
	require.Equal(t, "9001", lines[0].PlacementID)
}
