package features

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPResidentialIntelProvider_lookup(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/lookup", r.URL.Path)
		assert.Equal(t, "203.0.113.77", r.URL.Query().Get("ip"))
		assert.Equal(t, "secret", r.Header.Get("X-API-Key"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"residential_proxy":true,"vpn":true,"proxy":true}`))
	}))
	defer srv.Close()

	p, err := NewHTTPResidentialIntelProvider(srv.URL, "secret", 0)
	require.NoError(t, err)
	res, err := p.Lookup(context.Background(), "203.0.113.77")
	require.NoError(t, err)
	assert.True(t, res.IsResidentialProxyFarm())
}

func TestStubResidentialIntelProvider_lookup(t *testing.T) {
	p := &StubResidentialIntelProvider{
		Results: map[string]ResidentialIntelResult{
			"198.51.100.10": {ResidentialProxy: true},
		},
	}
	res, err := p.Lookup(context.Background(), "198.51.100.10")
	require.NoError(t, err)
	assert.True(t, res.ResidentialProxy)
}
