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

func TestFetchClickaduCosts_Httptest(t *testing.T) {
	customerID := uuid.New()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1.0/api/client/statistics/", r.URL.Path)
		require.Equal(t, "2026-08-24", r.URL.Query().Get("dateFrom"))
		require.Equal(t, "campaignId", r.URL.Query().Get("groupBy"))
		require.Equal(t, "clk-key", r.Header.Get("Authorization"))

		_ = json.NewEncoder(w).Encode(map[string]any{
			"result": []map[string]any{
				{"campaignId": 22001, "spent": 9.1},
			},
		})
	}))
	defer srv.Close()

	client := &http.Client{Transport: roundTripRewriteHost(srv.URL, nil)}
	cred := Credential{CustomerID: customerID, APIKey: "clk-key"}
	date := time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC)
	lines, err := fetchClickaduCosts(context.Background(), client, srv.URL+"/v1.0", cred, date)
	require.NoError(t, err)
	require.Len(t, lines, 1)
	require.Equal(t, "clickadu", lines[0].Network)
	require.Equal(t, int64(9_100_000), lines[0].AmountMicro)
	require.Equal(t, "22001", lines[0].PlacementID)
}
