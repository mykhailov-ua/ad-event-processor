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

func TestFetchExoClickCosts_Httptest(t *testing.T) {
	customerID := uuid.New()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/login":
			require.Equal(t, http.MethodPost, r.Method)
			var body map[string]string
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			require.Equal(t, "exo-api", body["api_token"])
			_ = json.NewEncoder(w).Encode(map[string]any{
				"token": "session-tok",
				"type":  "Bearer",
			})
		case "/v2/statistics/a/global":
			require.Equal(t, http.MethodPost, r.Method)
			require.Equal(t, "Bearer session-tok", r.Header.Get("Authorization"))
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			filter, ok := body["filter"].(map[string]any)
			require.True(t, ok)
			require.Equal(t, "2026-08-24", filter["date_from"])
			require.Equal(t, "2026-08-24", filter["date_to"])
			_ = json.NewEncoder(w).Encode(map[string]any{
				"result": []map[string]any{
					{
						"cost": 11.2,
						"group_by": map[string]any{
							"campaign_id": map[string]any{"id": "7788"},
						},
					},
				},
			})
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	client := &http.Client{Transport: roundTripRewriteHost(srv.URL, nil)}
	cred := Credential{
		CustomerID: customerID,
		APIKey:     "exo-api",
	}
	date := time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC)
	lines, err := fetchExoClickCosts(context.Background(), client, srv.URL+"/v2", cred, date)
	require.NoError(t, err)
	require.Len(t, lines, 1)
	require.Equal(t, "exoclick", lines[0].Network)
	require.Equal(t, int64(11_200_000), lines[0].AmountMicro)
	require.Equal(t, "7788", lines[0].PlacementID)
}
