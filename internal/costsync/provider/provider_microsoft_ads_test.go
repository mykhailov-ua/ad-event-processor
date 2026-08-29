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

func TestFetchMicrosoftAdsCosts_Httptest(t *testing.T) {
	customerID := uuid.New()
	reportCSV := "CampaignId,Spend\n\"99001\",\"12.50\"\n"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/Reporting/v13/GenerateReport/Submit":
			require.Equal(t, http.MethodPost, r.Method)
			require.Equal(t, "Bearer ms-tok", r.Header.Get("Authorization"))
			require.Equal(t, "cust-1", r.Header.Get("CustomerId"))
			require.Equal(t, "141234567", r.Header.Get("CustomerAccountId"))
			require.Equal(t, "dev-tok", r.Header.Get("DeveloperToken"))
			_ = json.NewEncoder(w).Encode(map[string]string{"ReportRequestId": "req-abc"})
		case "/Reporting/v13/GenerateReport/Poll":
			require.Equal(t, http.MethodPost, r.Method)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ReportRequestStatus": map[string]string{
					"Status":            "Success",
					"ReportDownloadUrl": "http://mock.local/report.csv",
				},
			})
		case "/report.csv":
			require.Equal(t, http.MethodGet, r.Method)
			_, _ = w.Write([]byte(reportCSV))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	client := &http.Client{Transport: roundTripRewriteHost(srv.URL, nil)}
	cred := Credential{
		CustomerID:  customerID,
		AccountID:   "141234567",
		AccessToken: "ms-tok",
		ExtraConfig: map[string]string{
			"customer_id":     "cust-1",
			"developer_token": "dev-tok",
		},
	}
	date := time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC)
	lines, err := fetchMicrosoftAdsCosts(context.Background(), client, srv.URL+"/Reporting/v13", cred, date)
	require.NoError(t, err)
	require.Len(t, lines, 1)
	require.Equal(t, "microsoft_ads", lines[0].Network)
	require.Equal(t, int64(12_500_000), lines[0].AmountMicro)
	require.Equal(t, "99001", lines[0].PlacementID)
}

func TestParseMicrosoftAdsCampaignCSV(t *testing.T) {
	customerID := uuid.New()
	cred := Credential{CustomerID: customerID}
	raw := []byte(`CampaignId,Spend
"123","3.25"
`)
	date := time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC)
	lines, err := parseMicrosoftAdsCampaignCSV(cred, date, raw)
	require.NoError(t, err)
	require.Len(t, lines, 1)
	require.Equal(t, int64(3_250_000), lines[0].AmountMicro)
}
