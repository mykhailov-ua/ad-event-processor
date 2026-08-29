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

func TestFetchAdsterraCosts_Httptest(t *testing.T) {
	customerID := uuid.New()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/advertiser/stats.json", r.URL.Path)
		require.Equal(t, "2026-08-24", r.URL.Query().Get("start_date"))
		require.Equal(t, "2026-08-24", r.URL.Query().Get("finish_date"))
		require.Equal(t, "campaign", r.URL.Query().Get("group_by"))
		require.Equal(t, "ads-key", r.Header.Get("X-API-Key"))

		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": []map[string]any{
				{"campaign_id": 44001, "spent": 6.75},
			},
		})
	}))
	defer srv.Close()

	client := &http.Client{Transport: roundTripRewriteHost(srv.URL, nil)}
	cred := Credential{
		CustomerID: customerID,
		APIKey:     "ads-key",
	}
	date := time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC)
	lines, err := fetchAdsterraCosts(context.Background(), client, srv.URL, cred, date)
	require.NoError(t, err)
	require.Len(t, lines, 1)
	require.Equal(t, "adsterra", lines[0].Network)
	require.Equal(t, int64(6_750_000), lines[0].AmountMicro)
	require.Equal(t, "44001", lines[0].PlacementID)
}
