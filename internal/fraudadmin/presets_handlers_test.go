package fraudadmin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ad-event-processor/pkg/httpresponse"

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
	h := &HTTPHandlers{
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
