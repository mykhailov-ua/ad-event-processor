package controlplane

import (
	"context"
	"fmt"

	"espx/internal/domain"

	"github.com/google/uuid"
)

const slotMigrationR5SamplePerShard = 3

func (s *Service) VerifySlotMigrationR5(ctx context.Context) error {
	if len(s.rdbs) == 0 {
		return fmt.Errorf("no redis shards configured")
	}
	campaignIDs, err := s.listActiveCampaignUUIDs(ctx)
	if err != nil {
		return err
	}
	if len(campaignIDs) == 0 {
		return nil
	}

	sharder := domain.NewStaticSlotSharder(len(s.rdbs))
	perShard := make(map[int][]uuid.UUID)
	for _, id := range campaignIDs {
		shard := sharder.GetShard(id)
		if len(perShard[shard]) < slotMigrationR5SamplePerShard {
			perShard[shard] = append(perShard[shard], id)
		}
	}

	for shard, ids := range perShard {
		if shard < 0 || shard >= len(s.rdbs) {
			continue
		}
		rdb := s.rdbs[shard]
		for _, campID := range ids {
			snap, err := domain.ReadBudgetInvariant(ctx, s.GetPool(), rdb, campID)
			if err != nil {
				return fmt.Errorf("r5 read shard %d campaign %s: %w", shard, campID, err)
			}
			spend := snap.BudgetLimit - snap.RedisRemaining
			expected := snap.PGCurrentSpend + snap.SyncDelta
			diff := spend - expected
			if diff < -1 || diff > 1 {
				return fmt.Errorf("r5 violated shard %d campaign %s: spend=%d expected=%d diff=%d",
					shard, campID, spend, expected, diff)
			}
		}
	}
	return nil
}

func (s *Service) HasPendingSlotDrain(ctx context.Context) (bool, error) {
	migRepo := domain.NewSlotMigrationRepo(s.GetPool())
	jobs, err := migRepo.ListDraining(ctx)
	if err != nil {
		return false, err
	}
	return len(jobs) > 0, nil
}
