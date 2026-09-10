package inbound

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerifyIP_allowlist(t *testing.T) {
	t.Parallel()
	cfg := AuthConfig{IPAllowlist: []string{"203.0.113.0/24"}}
	req := httptest.NewRequest("POST", "/track", nil)
	req.RemoteAddr = "203.0.113.44:1234"
	require.NoError(t, VerifyHTTP(cfg, req, nil))
}

func TestVerifyIP_rejects_holdout(t *testing.T) {
	t.Parallel()
	cfg := AuthConfig{IPAllowlist: []string{"203.0.113.0/24"}}
	req := httptest.NewRequest("POST", "/track", nil)
	req.RemoteAddr = "198.51.100.1:1234"
	assert.ErrorIs(t, VerifyHTTP(cfg, req, nil), ErrAuthFailed)
}
