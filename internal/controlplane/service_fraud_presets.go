package controlplane

import (
	"context"
	"errors"

	"ad-event-processor/internal/fraudadmin"
)

func mapFraudadminErr(err error) error {
	if errors.Is(err, fraudadmin.ErrValidation) {
		return errValidation(err.Error())
	}
	return err
}

func (s *Service) ListFraudPolicyPresets(ctx context.Context) ([]FraudPolicyPresetDTO, error) {
	return fraudadmin.ListPolicyPresets(ctx, s.GetPool())
}

func (s *Service) resolveFraudPresetThresholds(ctx context.Context, name string) (uint8, uint8, uint8, uint8, error) {
	pass, suspect, ivt, block, err := fraudadmin.ResolvePresetThresholds(ctx, s.GetPool(), name)
	if err != nil {
		return 0, 0, 0, 0, mapFraudadminErr(err)
	}
	return pass, suspect, ivt, block, nil
}

func (s *Service) UpdateFraudPolicyPreset(ctx context.Context, name string, req PatchFraudPolicyPresetRequest) (FraudPolicyPresetDTO, error) {
	out, err := fraudadmin.UpdatePolicyPreset(ctx, fraudPresetsHost{svc: s}, name, req)
	if err != nil {
		return FraudPolicyPresetDTO{}, mapFraudadminErr(err)
	}
	return out, nil
}
