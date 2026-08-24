package controlplane

import (
	"context"

	"ad-event-processor/internal/dedup"
)

func (s *Service) dedupAdapter(ctx context.Context) *dedup.Adapter {
	if s == nil || s.pool == nil || s.cfg == nil {
		return nil
	}
	epoch := dedup.LoadRoutingEpoch(ctx, s.pool)
	return dedup.NewAdapter(s.pool, s.cfg.RegionCode, epoch)
}
