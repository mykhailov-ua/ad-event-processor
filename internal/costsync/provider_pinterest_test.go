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

func TestFetchPinterestCosts_Httptest(t *testing.T) {
	customerID := uuid.New()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v5/ad_accounts/55846658/campaigns":
			require.Equal(t, http.MethodGet, r.Method)
			require.Equal(t, "Bearer pin-tok", r.Header.Get("Authorization"))
			_ = json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]string{{"id": "84772"}},
			})
		case "/v5/ad_accounts/55846658/campaigns/analytics":
			require.Equal(t, http.MethodGet, r.Method)
			require.Equal(t, "2026-08-24", r.URL.Query().Get("start_date"))
			require.Equal(t, "84772", r.URL.Query().Get("campaign_ids"))
			require.Equal(t, "SPEND_IN_DOLLAR", r.URL.Query().Get("columns"))
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{
					"DATE":            "2026-08-24",
					"CAMPAIGN_ID":     "84772",
					"SPEND_IN_DOLLAR": 9.5,
				},
			})
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	client := &http.Client{Transport: roundTripRewriteHost(srv.URL, nil)}
	cred := Credential{
		CustomerID:  customerID,
		AccountID:   "55846658",
		AccessToken: "pin-tok",
	}
	date := time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC)
	lines, err := fetchPinterestCosts(context.Background(), client, srv.URL+"/v5", cred, date)
	require.NoError(t, err)
	require.Len(t, lines, 1)
	require.Equal(t, "pinterest", lines[0].Network)
	require.Equal(t, int64(9_500_000), lines[0].AmountMicro)
	require.Equal(t, "84772", lines[0].PlacementID)
}
