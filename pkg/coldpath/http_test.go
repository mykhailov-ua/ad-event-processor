package coldpath

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePathUUID(t *testing.T) {
	id := uuid.NewString()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/campaigns/"+id+"/stats", http.NoBody)
	req.SetPathValue("id", id)
	got, err := ParsePathUUID(req, "id")
	require.NoError(t, err)
	assert.Equal(t, id, got.String())

	req.SetPathValue("id", "not-a-uuid")
	_, err = ParsePathUUID(req, "id")
	require.Error(t, err)
}

func TestPaginate(t *testing.T) {
	page, err := Paginate("", 10, 100)
	require.NoError(t, err)
	assert.Equal(t, 0, page.Offset)
	assert.Equal(t, 10, page.Limit)

	page, err = Paginate(EncodeCursor(25), 0, 100)
	require.NoError(t, err)
	assert.Equal(t, 25, page.Offset)
	assert.Equal(t, 50, page.Limit)

	_, err = Paginate("!!!", 10, 100)
	require.Error(t, err)
}

func TestReadLimitedBody_RejectsOversize(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(strings.Repeat("x", DefaultMaxBody+1)))
	rec := httptest.NewRecorder()

	_, err := ReadLimitedBody(rec, req, DefaultMaxBody)
	require.Error(t, err)
}

func TestDecodeRequestOrBadRequest(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"ok"}`))
	rec := httptest.NewRecorder()

	v, ok := DecodeRequestOrBadRequest[struct{ Name string }](rec, req, 1024)
	require.True(t, ok)
	assert.Equal(t, "ok", v.Name)

	req = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{`))
	rec = httptest.NewRecorder()
	_, ok = DecodeRequestOrBadRequest[struct{}](rec, req, 1024)
	require.False(t, ok)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
