package controlplane

import (
	ctrlhttp "ad-event-processor/internal/control/http"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ad-event-processor/internal/fraudadmin"
	"ad-event-processor/internal/opsadmin"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fraudPresetsGovernanceStub struct{}

func (s fraudPresetsGovernanceStub) ListFraudPolicyPresets(context.Context) ([]fraudadmin.FraudPolicyPresetDTO, error) {
	return nil, nil
}

func (s fraudPresetsGovernanceStub) UpdateFraudPolicyPreset(context.Context, string, fraudadmin.PatchFraudPolicyPresetRequest) (fraudadmin.FraudPolicyPresetDTO, error) {
	return fraudadmin.FraudPolicyPresetDTO{}, nil
}

func TestFraudPresetGovernance_supportCannotPatchGlobal_holdout(t *testing.T) {
	t.Parallel()
	assert.False(t, ctrlhttp.HasPermission(ctrlhttp.RoleSupport, ctrlhttp.PermShardsWrite))
	assert.False(t, ctrlhttp.HasPermission(ctrlhttp.RoleSupport, ctrlhttp.PermOpsWrite))
}

func TestPatchFraudPolicyPreset_supportRoleForbidden(t *testing.T) {
	t.Parallel()
	ops := &opsadmin.HTTPHandlers{
		FraudPresets: fraudPresetsGovernanceStub{},
		RequirePermission: func(perm string, next http.HandlerFunc) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				if !ctrlhttp.HasPermission(ctrlhttp.RoleSupport, perm) {
					http.Error(w, "forbidden", http.StatusForbidden)
					return
				}
				next(w, r)
			}
		},
		WriteServiceError: func(w http.ResponseWriter, err error) {
			status, code, msg := mapServiceError(err)
			http.Error(w, code+": "+msg, status)
		},
	}
	mux := http.NewServeMux()
	limit := func(next http.HandlerFunc) http.HandlerFunc { return next }
	perm := func(p string, next http.HandlerFunc) http.HandlerFunc {
		return ops.RequirePermission(p, next)
	}
	ops.RegisterFraudPresetOpsRoutes(mux, limit, perm)

	body := `{"pass":25,"suspect":55,"ivt":75,"block":95}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/ops/fraud/presets/aggressive", strings.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusForbidden, rec.Code)
}
