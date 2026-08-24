package controlplane

import (
	"context"
	"net/http"

	"ad-event-processor/pkg/httpresponse"
)

type FraudPresetsService interface {
	ListFraudPolicyPresets(ctx context.Context) ([]FraudPolicyPresetDTO, error)
	UpdateFraudPolicyPreset(ctx context.Context, name string, req PatchFraudPolicyPresetRequest) (FraudPolicyPresetDTO, error)
}

func (fraud *FraudHTTPHandlers) registerFraudPresetRoutes(mux *http.ServeMux, limit func(http.HandlerFunc) http.HandlerFunc, permAny func([]string, http.HandlerFunc) http.HandlerFunc) {
	if fraud == nil || fraud.Presets == nil {
		return
	}
	mux.HandleFunc("GET /api/v1/fraud/presets", limit(permAny([]string{
		"campaigns:read",
		"campaigns:read:masked",
		"audit:read",
		"shards:read",
	}, fraud.listFraudPresets)))
}

func (fraud *FraudHTTPHandlers) listFraudPresets(w http.ResponseWriter, r *http.Request) {
	presets, err := fraud.Presets.ListFraudPolicyPresets(r.Context())
	if err != nil {
		fraud.writeServiceError(w, err)
		return
	}
	if presets == nil {
		presets = []FraudPolicyPresetDTO{}
	}
	httpresponse.JSON(w, http.StatusOK, presets)
}

type fraudPresetsAPIAdapter struct {
	svc *Service
}

func (a fraudPresetsAPIAdapter) ListFraudPolicyPresets(ctx context.Context) ([]FraudPolicyPresetDTO, error) {
	return a.svc.ListFraudPolicyPresets(ctx)
}

func (a fraudPresetsAPIAdapter) UpdateFraudPolicyPreset(ctx context.Context, name string, req PatchFraudPolicyPresetRequest) (FraudPolicyPresetDTO, error) {
	return a.svc.UpdateFraudPolicyPreset(ctx, name, req)
}
