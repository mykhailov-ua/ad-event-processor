package coldpath

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAPIPagination_defaultsAndCaps(t *testing.T) {
	cases := []struct {
		query      string
		wantLimit  int32
		wantOffset int32
	}{
		{"", 50, 0},
		{"limit=10", 10, 0},
		{"limit=1000", 1000, 0},
		{"limit=5000", 1000, 0},
		{"offset=25", 50, 25},
		{"limit=10&offset=5", 10, 5},
		{"limit=0", 50, 0},
		{"limit=-1", 50, 0},
		{"offset=0", 50, 0},
	}

	for _, tc := range cases {
		t.Run(tc.query, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/api/v1/recon/runs?"+tc.query, http.NoBody)
			limit, offset := ParseAPIPagination(req)
			assert.Equal(t, tc.wantLimit, limit, "limit")
			assert.Equal(t, tc.wantOffset, offset, "offset")
		})
	}
}

func TestParseAPIPaginationWith_disputesCaps(t *testing.T) {
	req, _ := http.NewRequest("GET", "/api/v1/disputes?limit=500&offset=10", http.NoBody)
	limit, offset := ParseAPIPaginationWith(req, 20, 100)
	assert.Equal(t, int32(100), limit)
	assert.Equal(t, int32(10), offset)

	req, _ = http.NewRequest("GET", "/api/v1/disputes", http.NoBody)
	limit, offset = ParseAPIPaginationWith(req, 20, 100)
	assert.Equal(t, int32(20), limit)
	assert.Equal(t, int32(0), offset)
}

func TestParseCursorPagination_defaultsAndCaps(t *testing.T) {
	req, _ := http.NewRequest("GET", "/api/v1/reports/placements?limit=2000", http.NoBody)
	page, err := ParseCursorPagination(req, 10, 1000)
	require.NoError(t, err)
	assert.Equal(t, 1000, page.Limit)
	assert.Equal(t, 0, page.Offset)

	req, _ = http.NewRequest("GET", "/api/v1/reports/placements", http.NoBody)
	page, err = ParseCursorPagination(req, 50, 1000)
	require.NoError(t, err)
	assert.Equal(t, 50, page.Limit)
}
