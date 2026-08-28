package controlplane

import (
	"net/http"
	"net/http/httptest"
	"testing"

	ctrlhttp "ad-event-processor/internal/control/http"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCore_NormalizeRole(t *testing.T) {
	t.Parallel()
	assert.Equal(t, RoleAdmin, NormalizeRole("superadmin"))
	assert.Equal(t, RoleManager, NormalizeRole("manager"))
	assert.Equal(t, RoleUser, NormalizeRole("customer"))
}

func TestCore_HasPermission(t *testing.T) {
	t.Parallel()
	assert.True(t, HasPermission(RoleAdmin, "shards:write"))
	assert.False(t, HasPermission(RoleUser, "shards:write"))
	assert.True(t, HasPermission(RoleUser, "campaigns:read"))
}

func TestCore_clientIP(t *testing.T) {
	t.Parallel()
	r := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	r.RemoteAddr = "203.0.113.5:1234"
	r.Header.Set("X-Forwarded-For", "203.0.113.1, 198.51.100.2")
	assert.Equal(t, "203.0.113.5", ctrlhttp.ClientIP(r))

	r = httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	r.RemoteAddr = "192.0.2.1:1234"
	assert.Equal(t, "192.0.2.1", ctrlhttp.ClientIP(r))
}

func TestCore_mapNotFound(t *testing.T) {
	t.Parallel()
	assert.ErrorIs(t, mapNotFound(pgx.ErrNoRows, ErrCustomerNotFound), ErrCustomerNotFound)
	assert.NoError(t, mapNotFound(nil, ErrCustomerNotFound))
}

func TestCore_GetPermissionsForRole_unknown(t *testing.T) {
	t.Parallel()
	assert.Empty(t, GetPermissionsForRole("unknown"))
}

func TestCore_apiKeyRateLimiter(t *testing.T) {
	t.Parallel()
	lim := newAPIKeyRateLimiter(100, 10)
	assert.True(t, lim.Allow("digest-a"))
}

func TestCore_ipRateLimiter(t *testing.T) {
	t.Parallel()
	lim := newIPRateLimiter(100, 10)
	require.True(t, lim.Allow("1.2.3.4"))
}
