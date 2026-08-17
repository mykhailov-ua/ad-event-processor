package domainhealth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReputationChecker_SafeBrowsingFlagged(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		_ = json.NewEncoder(w).Encode(safeBrowsingResponse{
			Matches: []safeBrowsingMatch{{ThreatType: "MALWARE"}},
		})
	}))
	defer srv.Close()

	checker := NewReputationChecker(ReputationConfig{
		SafeBrowsingAPIKey: "test-key",
		SafeBrowsingAPIURL: srv.URL,
	})
	checker.http = srv.Client()

	unsafe, detail, err := checker.Check(context.Background(), "bad.example")
	require.NoError(t, err)
	require.True(t, unsafe)
	require.Contains(t, detail, "MALWARE")
}

func TestReputationChecker_FacebookBlocked(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Contains(t, r.URL.RawQuery, "scrape=true")
		_ = json.NewEncoder(w).Encode(facebookScrapeResponse{IsBlocked: true})
	}))
	defer srv.Close()

	checker := NewReputationChecker(ReputationConfig{
		FacebookToken:     "fb-token",
		FacebookGraphBase: srv.URL,
	})
	checker.http = srv.Client()

	unsafe, detail, err := checker.Check(context.Background(), "flagged.example")
	require.NoError(t, err)
	require.True(t, unsafe)
	require.Contains(t, detail, "is_blocked")
}

func TestReputationChecker_APIFailClosed(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("upstream error"))
	}))
	defer srv.Close()

	checker := NewReputationChecker(ReputationConfig{
		SafeBrowsingAPIKey: "key",
		SafeBrowsingAPIURL: srv.URL,
	})
	checker.http = srv.Client()

	_, _, err := checker.Check(context.Background(), "host.example")
	require.Error(t, err)
}

func TestNewReputationChecker_nilWithoutCredentials(t *testing.T) {
	t.Parallel()
	require.Nil(t, NewReputationChecker(ReputationConfig{}))
}
