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

func TestFetchMondiadCosts_Httptest(t *testing.T) {
	customerID := uuid.New()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/1.0/report/advertising/campaign":
			require.Equal(t, http.MethodGet, r.Method)
			require.Equal(t, "Bearer jwt-access", r.Header.Get("Authorization"))
			require.Equal(t, "CAMPAIGN", r.URL.Query().Get("breakdown"))
			require.Equal(t, "2026-08-24", r.URL.Query().Get("startDate"))
			_, _ = w.Write([]byte(`{
				"data": [
					{"campaignId": 901, "spent": 4.5, "clicks": 10}
				],
				"message": "Ok",
				"status": "OK"
			}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	client := &http.Client{Transport: roundTripRewriteHost(srv.URL, nil)}
	cred := Credential{
		CustomerID:  customerID,
		AccessToken: "jwt-access",
	}
	date := time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC)
	lines, err := fetchMondiadCosts(context.Background(), client, srv.URL, cred, date)
	require.NoError(t, err)
	require.Len(t, lines, 1)
	require.Equal(t, "mondiad", lines[0].Network)
	require.Equal(t, int64(4_500_000), lines[0].AmountMicro)
	require.Equal(t, "901", lines[0].PlacementID)
}

func TestOAuthRefresh_MondiadHttptest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/1.0/auth/login", r.URL.Path)
		var body map[string]string
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.Equal(t, "mond-client", body["clientId"])
		require.Equal(t, "mond-secret", body["clientSecret"])
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"token":           "mond-jwt",
				"refreshToken":    "mond-refresh",
				"durationSeconds": 120,
			},
			"message": "Ok",
			"status":  "OK",
		})
	}))
	defer srv.Close()

	client := &http.Client{Transport: roundTripRewriteHost(srv.URL, nil)}
	token, refresh, expires, err := RefreshMondiadOAuth(context.Background(), client, srv.URL, Credential{
		ExtraConfig: map[string]string{"client_id": "mond-client"},
		APIKey:      "mond-secret",
	})
	require.NoError(t, err)
	require.Equal(t, "mond-jwt", token)
	require.Equal(t, "mond-refresh", refresh)
	require.True(t, expires.After(time.Now()))
}
