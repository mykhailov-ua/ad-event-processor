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

func TestFetchHilltopAdsCosts_Httptest(t *testing.T) {
	customerID := uuid.New()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/advertiser/listStats", r.URL.Path)
		require.Equal(t, "hta-key", r.URL.Query().Get("key"))
		require.Equal(t, "2026-08-24", r.URL.Query().Get("date"))
		require.Equal(t, "campaignID", r.URL.Query().Get("group"))

		_ = json.NewEncoder(w).Encode(map[string]any{
			"result": []map[string]any{
				{"campaignID": "9911", "spent": 4.2},
			},
		})
	}))
	defer srv.Close()

	client := &http.Client{Transport: roundTripRewriteHost(srv.URL, nil)}
	cred := Credential{CustomerID: customerID, APIKey: "hta-key"}
	date := time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC)
	lines, err := fetchHilltopAdsCosts(context.Background(), client, srv.URL, cred, date)
	require.NoError(t, err)
	require.Len(t, lines, 1)
	require.Equal(t, "hilltopads", lines[0].Network)
	require.Equal(t, int64(4_200_000), lines[0].AmountMicro)
	require.Equal(t, "9911", lines[0].PlacementID)
}
