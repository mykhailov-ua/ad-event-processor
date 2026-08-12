package controlplane

import (
	"context"
	"fmt"

	"github.com/bidshard/ad-event-processor/internal/licensing"
)

func (s *Service) enforceDeploymentLicenseCampaignCap(ctx context.Context) error {
	if s == nil || s.GetPool() == nil {
		return nil
	}
	limits, state, ok := licenseDeploymentLimits()
	if !ok {
		return nil
	}
	if !licensing.IngestAllowed(state) {
		return errValidation("license not active")
	}
	max := limits.MaxActiveCampaigns
	if max == 0 {
		return nil
	}
	var active int64
	err := s.GetPool().QueryRow(ctx, `
		SELECT COUNT(*) FROM campaigns WHERE status = 'ACTIVE'`).Scan(&active)
	if err != nil {
		return fmt.Errorf("count deployment active campaigns: %w", err)
	}
	if uint64(active) >= max {
		return ErrDeploymentCampaignLimit
	}
	return nil
}
