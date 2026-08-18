package controlplane

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bidshard/ad-event-processor/pkg/httpresponse"
	"github.com/stretchr/testify/require"
)

type fraudPresetsStub struct {
	list []FraudPolicyPresetDTO
	last PatchFraudPolicyPresetRequest
	name string
}

func (s *fraudPresetsStub) ListFraudPolicyPresets(_ context.Context) ([]FraudPolicyPresetDTO, error) {
	return s.list, nil
}

func (s *fraudPresetsStub) UpdateFraudPolicyPreset(_ context.Context, name string, req PatchFraudPolicyPresetRequest) (FraudPolicyPresetDTO, error) {
	s.name = name
	s.last = req
	return FraudPolicyPresetDTO{Name: name, Pass: 25, Suspect: 55, IVT: 75, Block: 95}, nil
}

func TestListFraudPresetsHandler_returnsRows(t *testing.T) {
	stub := &fraudPresetsStub{
		list: []FraudPolicyPresetDTO{{Name: "balanced", Pass: 30, Suspect: 60, IVT: 80, Block: 100}},
	}
	h := &FraudHTTPHandlers{
		Presets: stub,
		RequireAnyPermission: func(_ []string, next http.HandlerFunc) http.HandlerFunc {
			return next
		},
		WriteServiceError: func(w http.ResponseWriter, _ error) {
			httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL", "internal")
		},
	}
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/fraud/presets", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var out []FraudPolicyPresetDTO
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	require.Len(t, out, 1)
}

func TestPatchFraudPolicyPresetHandler_updatesPreset(t *testing.T) {
	stub := &fraudPresetsStub{}
	ops := &OpsHTTPHandlers{
		FraudPresets: stub,
		RequirePermission: func(_ string, next http.HandlerFunc) http.HandlerFunc {
			return next
		},
		WriteServiceError: func(w http.ResponseWriter, _ error) {
			httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL", "internal")
		},
	}
	mux := http.NewServeMux()
	limit := func(next http.HandlerFunc) http.HandlerFunc { return next }
	perm := func(_ string, next http.HandlerFunc) http.HandlerFunc { return next }
	ops.registerFraudPresetOpsRoutes(mux, limit, perm)

	body := `{"pass":25,"suspect":55,"ivt":75,"block":95}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/ops/fraud/presets/aggressive", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "aggressive", stub.name)
}
