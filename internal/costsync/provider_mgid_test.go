package costsync

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestFetchMGIDCosts_Httptest(t *testing.T) {
	customerID := uuid.New()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Contains(t, r.URL.Path, "/v1/goodhits/clients/client-77/campaigns-stat")
		require.Equal(t, "interval", r.URL.Query().Get("dateInterval"))
		require.Equal(t, "2026-08-24", r.URL.Query().Get("startDate"))
		require.Equal(t, "2026-08-24", r.URL.Query().Get("endDate"))
		require.Equal(t, "Bearer mgid-token", r.Header.Get("Authorization"))

		_, _ = w.Write([]byte(`{
			"555001": {"campaign_id": 555001, "imps": 1000, "clicks": 40, "spent": 15.5, "avcpc": 0.3875}
		}`))
	}))
	defer srv.Close()

	client := &http.Client{Transport: roundTripRewriteHost(srv.URL, nil)}
	cred := Credential{
		CustomerID:  customerID,
		AccountID:   "client-77",
		AccessToken: "mgid-token",
	}
	date := time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC)
	lines, err := fetchMGIDCosts(context.Background(), client, srv.URL+"/v1", cred, date)
	require.NoError(t, err)
	require.Len(t, lines, 1)
	require.Equal(t, "mgid", lines[0].Network)
	require.Equal(t, int64(15_500_000), lines[0].AmountMicro)
	require.Equal(t, "555001", lines[0].PlacementID)
}
