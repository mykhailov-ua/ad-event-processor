package controlplane

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseListSort_defaultsAndValid(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/customers?sort=name&order=asc", http.NoBody)
	field, order, err := parseListSort(req, map[string]struct{}{"name": {}, "created_at": {}}, "created_at")
	require.NoError(t, err)
	assert.Equal(t, "name", field)
	assert.Equal(t, "asc", order)
}

func TestParseListSort_invalidSort(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/customers?sort=evil", http.NoBody)
	_, _, err := parseListSort(req, map[string]struct{}{"name": {}}, "name")
	require.Error(t, err)
}

func TestParseListSort_invalidOrder(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/customers?sort=name&order=sideways", http.NoBody)
	_, _, err := parseListSort(req, map[string]struct{}{"name": {}}, "name")
	require.Error(t, err)
}

func TestFiltersAppliedFromQuery_echoesOnlySet(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/campaigns?q=foo&status=ACTIVE", http.NoBody)
	got := filtersAppliedFromQuery(req, "q", "status", "pacing_mode")
	assert.Equal(t, map[string]string{"q": "foo", "status": "ACTIVE"}, got)
}
