package trafficoptimizer

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPHandlers_listPresets(t *testing.T) {
	t.Parallel()
	h := &HTTPHandlers{Rules: &RulesService{}}
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/traffic-optimizer/presets", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var presets []Preset
	require.NoError(t, decodeJSON(rec.Body.Bytes(), &presets))
	require.Len(t, presets, 4)
	keys := make([]string, len(presets))
	for i, p := range presets {
		keys[i] = p.Key
	}
	assert.Contains(t, keys, "cr_best_performer")
	assert.Contains(t, keys, "epc_best_performer")
}

func TestHTTPHandlers_createRule_unavailableWithoutPool(t *testing.T) {
	t.Parallel()
	h := &HTTPHandlers{Rules: &RulesService{}}
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/traffic-optimizer/rules", jsonBody(t, UpsertRuleRequest{
		CustomerID: "00000000-0000-4000-8000-000000000001",
		Enabled:    true,
	}))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
}

func TestExpandPreset_crBestPerformer(t *testing.T) {
	t.Parallel()
	out, err := ExpandPreset("cr_best_performer", nil)
	require.NoError(t, err)
	assert.Equal(t, ObjectiveCR, out.Objective)
	assert.Equal(t, AlgorithmThompson, out.Algorithm)
	assert.Equal(t, ScopeLander, out.Scope)
}

func TestExpandPreset_unknown(t *testing.T) {
	t.Parallel()
	_, err := ExpandPreset("missing", nil)
	require.Error(t, err)
}

func TestApplyPreset_fillsFromPreset(t *testing.T) {
	t.Parallel()
	out, err := ApplyPreset(UpsertRuleRequest{
		PresetKey:  "epc_best_performer",
		CustomerID: "00000000-0000-4000-8000-000000000001",
		Enabled:    true,
	})
	require.NoError(t, err)
	assert.Equal(t, ObjectiveEPC, out.Objective)
	assert.Equal(t, "EPC best performer", out.Name)
}
