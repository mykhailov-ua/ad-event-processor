package controlplane

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ad-event-processor/internal/fraudadmin"
	"ad-event-processor/internal/opsadmin"
	"ad-event-processor/pkg/httpresponse"

	"github.com/stretchr/testify/require"
)

type fraudPresetsOpsStub struct {
	name string
	last fraudadmin.PatchFraudPolicyPresetRequest
}

func (s *fraudPresetsOpsStub) ListFraudPolicyPresets(_ context.Context) ([]fraudadmin.FraudPolicyPresetDTO, error) {
	return nil, nil
}

func (s *fraudPresetsOpsStub) UpdateFraudPolicyPreset(_ context.Context, name string, req fraudadmin.PatchFraudPolicyPresetRequest) (fraudadmin.FraudPolicyPresetDTO, error) {
	s.name = name
	s.last = req
	return fraudadmin.FraudPolicyPresetDTO{Name: name, Pass: 25, Suspect: 55, IVT: 75, Block: 95}, nil
}

func TestPatchFraudPolicyPresetHandler_updatesPreset(t *testing.T) {
	stub := &fraudPresetsOpsStub{}
	ops := &opsadmin.HTTPHandlers{
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
	ops.RegisterFraudPresetOpsRoutes(mux, limit, perm)

	body := `{"pass":25,"suspect":55,"ivt":75,"block":95}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/ops/fraud/presets/aggressive", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "aggressive", stub.name)
}
