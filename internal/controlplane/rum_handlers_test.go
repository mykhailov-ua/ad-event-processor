package controlplane

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubRUMStore struct {
	events []ClientRUMIngestDTO
}

func (s *stubRUMStore) AppendClientRUM(ev ClientRUMIngestDTO) {
	s.events = append(s.events, ev)
}

func (s *stubRUMStore) SnapshotClientRUM() []any {
	out := make([]any, len(s.events))
	for i, ev := range s.events {
		out[i] = ev
	}
	return out
}

func TestPostClientRUM_Accepted(t *testing.T) {
	t.Parallel()
	store := &stubRUMStore{}
	h := &OpsHTTPHandlers{RUMStore: store}
	h.registerRUMRoutes(http.NewServeMux())

	body := `{"path":"/campaigns","api":{"slowPaths":[]}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ops/rum", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.postClientRUM(rec, req)
	require.Equal(t, http.StatusAccepted, rec.Code)
	require.Len(t, store.events, 1)
	assert.Equal(t, "/campaigns", store.events[0].Path)
}

func TestGetClientRUM_OK(t *testing.T) {
	t.Parallel()
	store := &stubRUMStore{}
	store.AppendClientRUM(ClientRUMIngestDTO{Path: "/ops"})
	h := &OpsHTTPHandlers{
		RUMStore: store,
		RequirePermission: func(_ string, next http.HandlerFunc) http.HandlerFunc {
			return next
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/ops/rum", http.NoBody)
	rec := httptest.NewRecorder()
	h.getClientRUM(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Events []any `json:"events"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Len(t, resp.Events, 1)
}
