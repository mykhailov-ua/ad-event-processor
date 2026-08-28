package platformadmin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testObserver struct {
	results []struct {
		vendor  string
		success bool
		latency time.Duration
	}
	errors []string
}

func (o *testObserver) ObserveProbe(vendor string, success bool, latency time.Duration) {
	o.results = append(o.results, struct {
		vendor  string
		success bool
		latency time.Duration
	}{vendor, success, latency})
}

func (o *testObserver) ObserveProbeError(vendor string) {
	o.errors = append(o.errors, vendor)
}

func TestMaxMindProbe_localFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "GeoLite2-Country.mmdb")
	require.NoError(t, os.WriteFile(path, []byte("mmdb"), 0o600))

	p := NewMaxMindProbe(path)
	require.NoError(t, p.Probe(context.Background()))

	require.NoError(t, os.WriteFile(path, nil, 0o600))
	assert.Error(t, p.Probe(context.Background()))
}

func TestStripeProbe_successAndFail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/balance" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer sk_test" {
			http.Error(w, "auth", http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"object":"balance"}`))
	}))
	defer srv.Close()

	old := stripeBalanceURL
	stripeBalanceURL = srv.URL + "/v1/balance"
	defer func() { stripeBalanceURL = old }()

	p := NewStripeProbe("sk_test", srv.Client())
	require.NoError(t, p.Probe(context.Background()))

	p2 := NewStripeProbe("sk_bad", srv.Client())
	assert.Error(t, p2.Probe(context.Background()))
}

func TestTelegramProbe_success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/bottok/getMe" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	old := telegramAPIBase
	telegramAPIBase = srv.URL + "/bot"
	defer func() { telegramAPIBase = old }()

	p := NewTelegramProbe("tok", srv.Client())
	require.NoError(t, p.Probe(context.Background()))
}

func TestWorker_observesTransitions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "db.mmdb")
	require.NoError(t, os.WriteFile(path, []byte("x"), 0o600))

	reg := NewRegistry()
	reg.Register(NewMaxMindProbe(path))

	obs := &testObserver{}
	w := NewWorker(reg, WorkerConfig{Interval: time.Hour, Timeout: time.Second}, obs)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go w.Start(ctx)
	w.runOnce(ctx)

	require.NotEmpty(t, obs.results)
	assert.True(t, obs.results[0].success)
	assert.Equal(t, "maxmind", obs.results[0].vendor)
}
