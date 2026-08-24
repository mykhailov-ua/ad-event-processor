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

func TestFetchGalaksionCosts_Httptest(t *testing.T) {
	customerID := uuid.New()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/api/v1/advertiser/statistics", r.URL.Path)
		require.Equal(t, "gal-token", r.Header.Get("X-Auth-Token"))
		require.Contains(t, r.URL.Query().Get("groupBy"), "campaign")
		require.Equal(t, "2026-08-24 00:00:00", r.URL.Query().Get("dateFrom"))
		require.Equal(t, "2026-08-24 23:59:59", r.URL.Query().Get("dateTo"))
		_ = json.NewEncoder(w).Encode(map[string]any{
			"rows": []map[string]any{
				{"campaign": 555, "money": 4.5, "impressions": 100},
			},
		})
	}))
	defer srv.Close()

	client := &http.Client{Transport: roundTripRewriteHost(srv.URL, nil)}
	cred := Credential{
		CustomerID:  customerID,
		AccessToken: "gal-token",
	}
	date := time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC)
	lines, err := fetchGalaksionCosts(context.Background(), client, srv.URL+"/api", cred, date)
	require.NoError(t, err)
	require.Len(t, lines, 1)
	require.Equal(t, "galaksion", lines[0].Network)
	require.Equal(t, int64(4_500_000), lines[0].AmountMicro)
	require.Equal(t, "555", lines[0].PlacementID)
}

func TestFetchGalaksionCosts_Login(t *testing.T) {
	customerID := uuid.New()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth":
			require.Equal(t, http.MethodPost, r.Method)
			var body map[string]string
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			require.Equal(t, "adv@example.com", body["email"])
			require.Equal(t, "secret", body["password"])
			_ = json.NewEncoder(w).Encode(map[string]string{"token": "gal-token"})
		case "/api/v1/advertiser/statistics":
			require.Equal(t, "gal-token", r.Header.Get("X-Auth-Token"))
			_ = json.NewEncoder(w).Encode(map[string]any{
				"rows": []map[string]any{{"campaign": 9, "money": 2.0}},
			})
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	client := &http.Client{Transport: roundTripRewriteHost(srv.URL, nil)}
	cred := Credential{
		CustomerID: customerID,
		AccountID:  "adv@example.com",
		ExtraConfig: map[string]string{
			"password": "secret",
		},
	}
	date := time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC)
	lines, err := fetchGalaksionCosts(context.Background(), client, srv.URL+"/api", cred, date)
	require.NoError(t, err)
	require.Len(t, lines, 1)
	require.Equal(t, "9", lines[0].PlacementID)
}
