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

func TestFetchJuicyAdsCosts_Httptest(t *testing.T) {
	customerID := uuid.New()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/campaigns/popunders/juicy-token":
			_, _ = w.Write([]byte(`[{"id":"55","campaign_name":"Camp"}]`))
		case "/statistics/popunders/advertiser/juicy-token/55/2026-08-24/2026-08-24":
			_, _ = w.Write([]byte(`[{"thedate":"2026-08-24","spend":"12.50","imps":"100"}]`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	client := &http.Client{Transport: roundTripRewriteHost(srv.URL, nil)}
	cred := Credential{
		CustomerID: customerID,
		APIKey:     "juicy-token",
	}
	date := time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC)
	lines, err := fetchJuicyAdsCosts(context.Background(), client, srv.URL, cred, date)
	require.NoError(t, err)
	require.Len(t, lines, 1)
	require.Equal(t, "juicyads", lines[0].Network)
	require.Equal(t, int64(12_500_000), lines[0].AmountMicro)
	require.Equal(t, "55", lines[0].PlacementID)
}
