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

func TestFetchRevcontentCosts_Httptest(t *testing.T) {
	customerID := uuid.New()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/stats/api/v1.0/boosts", r.URL.Path)
		require.Equal(t, "Bearer rc-token", r.Header.Get("Authorization"))
		require.Equal(t, "2026-08-24", r.URL.Query().Get("date_from"))
		require.Equal(t, "2026-08-24", r.URL.Query().Get("date_to"))

		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data": []map[string]any{
				{"id": "88001", "cost": "12.50"},
			},
		})
	}))
	defer srv.Close()

	client := &http.Client{Transport: roundTripRewriteHost(srv.URL, nil)}
	cred := Credential{CustomerID: customerID, AccessToken: "rc-token"}
	date := time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC)
	lines, err := fetchRevcontentCosts(context.Background(), client, srv.URL, cred, date)
	require.NoError(t, err)
	require.Len(t, lines, 1)
	require.Equal(t, "revcontent", lines[0].Network)
	require.Equal(t, int64(12_500_000), lines[0].AmountMicro)
	require.Equal(t, "88001", lines[0].PlacementID)
}
