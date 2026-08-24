package costsync

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

func TestFetchTrafficStarsCosts_Httptest(t *testing.T) {
	customerID := uuid.New()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v2/campaigns/statistics", r.URL.Path)
		require.Equal(t, "Bearer ts-access", r.Header.Get("Authorization"))
		var body map[string]string
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.Equal(t, "2026-08-24", body["date_from"])
		require.Equal(t, "2026-08-24", body["date_to"])
		require.Equal(t, "campaign", body["group_by"])
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": 9001, "costs": 12.75},
			},
		})
	}))
	defer srv.Close()

	client := &http.Client{Transport: roundTripRewriteHost(srv.URL, nil)}
	cred := Credential{
		CustomerID:  customerID,
		AccessToken: "ts-access",
	}
	date := time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC)
	lines, err := fetchTrafficStarsCosts(context.Background(), client, srv.URL, cred, date)
	require.NoError(t, err)
	require.Len(t, lines, 1)
	require.Equal(t, "trafficstars", lines[0].Network)
	require.Equal(t, int64(12_750_000), lines[0].AmountMicro)
	require.Equal(t, "9001", lines[0].PlacementID)
}

func TestFetchTrafficStarsCosts_OfflineKeyLogin(t *testing.T) {
	customerID := uuid.New()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/auth/token":
			require.Equal(t, http.MethodPost, r.Method)
			_ = r.ParseForm()
			require.Equal(t, "refresh_token", r.FormValue("grant_type"))
			require.Equal(t, "offline-key", r.FormValue("refresh_token"))
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "ts-access",
				"expires_in":   3600,
			})
		case "/v2/campaigns/statistics":
			require.Equal(t, "Bearer ts-access", r.Header.Get("Authorization"))
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{{"id": 42, "costs": 1.0}},
			})
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	client := &http.Client{Transport: roundTripRewriteHost(srv.URL, nil)}
	cred := Credential{
		CustomerID:   customerID,
		RefreshToken: "offline-key",
	}
	date := time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC)
	lines, err := fetchTrafficStarsCosts(context.Background(), client, srv.URL, cred, date)
	require.NoError(t, err)
	require.Len(t, lines, 1)
	require.Equal(t, "42", lines[0].PlacementID)
}
