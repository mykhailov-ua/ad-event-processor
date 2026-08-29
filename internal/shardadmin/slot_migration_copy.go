package shardadmin

import (
	"context"
	"fmt"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/jackc/pgx/v5/pgxpool"
)

func GetSlotMigrations(ctx context.Context, pool *pgxpool.Pool, version int32) ([]SlotMigrationDTO, error) {
	repo := domain.NewSlotMigrationRepo(pool)
	rows, err := repo.ListByVersion(ctx, version)
	if err != nil {
		return nil, err
	}
	out := make([]SlotMigrationDTO, 0, len(rows))
	for _, row := range rows {
		dto := SlotMigrationDTO{
			Version:         row.Version,
			Slot:            row.Slot,
			SourceShard:     row.SourceShard,
			TargetShard:     row.TargetShard,
			State:           string(row.State),
			CampaignsTotal:  row.CampaignsTotal,
			CampaignsCopied: row.CampaignsCopied,
		}
		if row.LastError.Valid {
			dto.LastError = row.LastError.String
		}
		out = append(out, dto)
	}
	return out, nil
}

func EnsureSlotMigrationJobs(ctx context.Context, host Host, draftVersion int32) error {
	mapRepo := domain.NewSlotMapRepo(host.Pool())
	migRepo := domain.NewSlotMigrationRepo(host.Pool())

	active, err := mapRepo.GetActiveVersion(ctx)
	if err != nil {
		return err
	}
	if draftVersion <= active {
		return fmt.Errorf("draft version %d must be greater than active %d", draftVersion, active)
	}

	activeRows, err := mapRepo.ListVersion(ctx, active)
	if err != nil {
		return err
	}
	sourceBySlot := make(map[int16]int16, len(activeRows))
	for _, row := range activeRows {
		sourceBySlot[row.Slot] = row.ShardID
	}

	migrating, err := mapRepo.ListMigratingSlots(ctx, draftVersion)
	if err != nil {
		return err
	}
	for _, row := range migrating {
		source, ok := sourceBySlot[row.Slot]
		if !ok {
			return fmt.Errorf("slot %d missing in active map", row.Slot)
		}
		if source == row.ShardID {
			return fmt.Errorf("slot %d source and target shard are both %d", row.Slot, source)
		}
		if err := migRepo.InsertIfAbsent(ctx, draftVersion, row.Slot, source, row.ShardID); err != nil {
			return err
		}
	}
	return nil
}

func CopySlotMigrationData(ctx context.Context, host Host, version int32, slot int16) error {
	if len(host.RedisShards()) == 0 {
		return fmt.Errorf("no redis shards configured")
	}
	migRepo := domain.NewSlotMigrationRepo(host.Pool())
	job, err := migRepo.Get(ctx, version, slot)
	if err != nil {
		return err
	}
	if job.State == db.RedisSlotMigrationStateCopied ||
		job.State == db.RedisSlotMigrationStateDualWriting ||
		job.State == db.RedisSlotMigrationStateDraining ||
		job.State == db.RedisSlotMigrationStateDone {
		return nil
	}
	if job.SourceShard < 0 || int(job.SourceShard) >= len(host.RedisShards()) ||
		job.TargetShard < 0 || int(job.TargetShard) >= len(host.RedisShards()) {
		return fmt.Errorf("invalid shard indices source=%d target=%d", job.SourceShard, job.TargetShard)
	}

	campaignIDs, err := host.ListActiveCampaignUUIDs(ctx)
	if err != nil {
		return err
	}
	slotCampaigns := domain.FilterCampaignIDsBySlot(campaignIDs, slot)
	total := int32(len(slotCampaigns))

	if err := migRepo.UpdateProgress(ctx, version, slot, total, job.CampaignsCopied,
		db.RedisSlotMigrationStateCopying, ""); err != nil {
		return err
	}

	src := host.RedisShards()[job.SourceShard]
	dst := host.RedisShards()[job.TargetShard]

	if host.MigrationFenceEnabled() && !host.SlotMigrationDualWriteEnabled() && len(slotCampaigns) > 0 {
		if err := domain.BumpMigrationFences(ctx, host.Pool(), src, slotCampaigns); err != nil {
			_ = migRepo.UpdateProgress(ctx, version, slot, total, job.CampaignsCopied,
				db.RedisSlotMigrationStateFailed, err.Error())
			return fmt.Errorf("migration fence: %w", err)
		}
	}

	migrator := &domain.CampaignKeyMigrator{}
	var copied int32
	for _, id := range slotCampaigns {
		if _, err := migrator.MigrateCampaignKeys(ctx, src, dst, id); err != nil {
			_ = migRepo.UpdateProgress(ctx, version, slot, total, copied,
				db.RedisSlotMigrationStateFailed, err.Error())
			return fmt.Errorf("copy campaign %s: %w", id, err)
		}
		copied++
		if copied%10 == 0 || copied == total {
			if err := migRepo.UpdateProgress(ctx, version, slot, total, copied,
				db.RedisSlotMigrationStateCopying, ""); err != nil {
				return err
			}
		}
	}

	catalog := domain.DefaultCampaignRedisKeyCatalog
	if err := verifyActivationKeysPostCopy(ctx, src, dst, catalog, slotCampaigns); err != nil {
		_ = migRepo.UpdateProgress(ctx, version, slot, total, copied,
			db.RedisSlotMigrationStateFailed, err.Error())
		return err
	}

	finalState := db.RedisSlotMigrationStateCopied
	if host.SlotMigrationDualWriteEnabled() {
		if err := domain.EnableSlotMigrationDualWrite(ctx, src, version, slot, job.TargetShard); err != nil {
			_ = migRepo.UpdateProgress(ctx, version, slot, total, copied,
				db.RedisSlotMigrationStateFailed, err.Error())
			return fmt.Errorf("enable dual-write slot %d: %w", slot, err)
		}
		finalState = db.RedisSlotMigrationStateDualWriting
	}

	return migRepo.UpdateProgress(ctx, version, slot, total, copied,
		finalState, "")
}

func CopyAllMigratingSlots(ctx context.Context, host Host, draftVersion int32) error {
	if err := EnsureSlotMigrationJobs(ctx, host, draftVersion); err != nil {
		return err
	}
	mapRepo := domain.NewSlotMapRepo(host.Pool())
	migrating, err := mapRepo.ListMigratingSlots(ctx, draftVersion)
	if err != nil {
		return err
	}
	for _, row := range migrating {
		if err := CopySlotMigrationData(ctx, host, draftVersion, row.Slot); err != nil {
			return fmt.Errorf("slot migration copy version=%d slot=%d: %w", draftVersion, row.Slot, err)
		}
	}
	return nil
}
