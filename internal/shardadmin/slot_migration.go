package shardadmin

import (
	"context"
	"fmt"
	"time"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/metrics"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

const slotMigrationR5SamplePerShard = 3

type SlotMigrationOrchestrator struct {
	host     Host
	interval time.Duration
}

func NewSlotMigrationOrchestrator(host Host, interval time.Duration) *SlotMigrationOrchestrator {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	return &SlotMigrationOrchestrator{host: host, interval: interval}
}

func (o *SlotMigrationOrchestrator) Start(ctx context.Context) {
	o.bumpPendingMigrationFences(ctx)

	ticker := time.NewTicker(o.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			o.tick(ctx)
		}
	}
}

func (o *SlotMigrationOrchestrator) tick(ctx context.Context) {
	migRepo := domain.NewSlotMigrationRepo(o.host.Pool())
	draft, err := migRepo.GetMaxDraftVersionWithMigrating(ctx)
	if err != nil {
		o.host.AlertSlotMigrationError(ctx, "draft_lookup", err)
		return
	}
	if draft > 0 {
		if err := CatchUpDualWriteSlots(ctx, o.host, draft); err != nil {
			o.host.AlertSlotMigrationError(ctx, "dual_write_catchup", err)
		}
		if err := CopyAllMigratingSlots(ctx, o.host, draft); err != nil {
			o.host.AlertSlotMigrationError(ctx, "copy", err)
		}
	}

	mapRepo := domain.NewSlotMapRepo(o.host.Pool())
	active, err := mapRepo.GetActiveVersion(ctx)
	if err != nil {
		o.host.AlertSlotMigrationError(ctx, "active_lookup", err)
		return
	}
	if err := DrainMigratingSlots(ctx, o.host, active); err != nil {
		o.host.AlertSlotMigrationError(ctx, "drain", err)
	} else {
		pending, pendErr := HasPendingSlotDrain(ctx, o.host.Pool())
		if pendErr == nil && !pending {
			if r5Err := VerifySlotMigrationR5(ctx, o.host); r5Err != nil {
				o.host.AlertSlotMigrationError(ctx, "r5_verify", r5Err)
			}
		}
	}
	o.host.CheckStuckDrainJobs(ctx)
}

func (o *SlotMigrationOrchestrator) bumpPendingMigrationFences(ctx context.Context) {
	if err := BumpFencesForPendingMigrations(ctx, o.host); err != nil {
		o.host.AlertSlotMigrationError(ctx, "bump_fences", err)
	}
}

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

func ActivateSlotMapVersionWithMigration(ctx context.Context, host Host, adminID uuid.UUID, version int32) error {
	mapRepo := domain.NewSlotMapRepo(host.Pool())
	migRepo := domain.NewSlotMigrationRepo(host.Pool())

	migrating, err := mapRepo.ListMigratingSlots(ctx, version)
	if err != nil {
		return err
	}
	campaignIDs, err := host.ListActiveCampaignUUIDs(ctx)
	if err != nil {
		return err
	}
	catalog := domain.DefaultCampaignRedisKeyCatalog

	if len(migrating) > 0 {
		if err := EnsureSlotMigrationJobs(ctx, host, version); err != nil {
			return err
		}
		for _, row := range migrating {
			job, err := migRepo.Get(ctx, version, row.Slot)
			if err != nil {
				return err
			}
			skipRewarm := false
			if job.State == db.RedisSlotMigrationStateDualWriting {
				if err := finalizeDualWriteSlot(ctx, host, version, job, campaignIDs); err != nil {
					return err
				}
				skipRewarm = true
				job, err = migRepo.Get(ctx, version, row.Slot)
				if err != nil {
					return err
				}
			}
			if job.State != db.RedisSlotMigrationStateCopied {
				return ErrSlotMigrationNotReady
			}
			if job.TargetShard < 0 || int(job.TargetShard) >= len(host.RedisShards()) {
				return fmt.Errorf("invalid target shard %d for slot %d", job.TargetShard, row.Slot)
			}
			slotCampaigns := domain.FilterCampaignIDsBySlot(campaignIDs, row.Slot)
			dst := host.RedisShards()[job.TargetShard]
			if !skipRewarm {
				if err := domain.RewarmCampaignBudgetKeys(ctx, host.Pool(), dst, slotCampaigns); err != nil {
					return fmt.Errorf("pg re-warm slot %d: %w", row.Slot, err)
				}
			}
			if err := catalog.VerifySlotCampaignKeysExist(ctx, dst, slotCampaigns); err != nil {
				metrics.SlotMigrationCutoverBlockedTotal.WithLabelValues("missing_keys").Inc()
				return fmt.Errorf("%w: %w", ErrSlotMigrationKeysMissing, err)
			}
		}
	}

	tx, err := host.Pool().Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := db.New(tx)
	meta, err := q.LockSlotMapMeta(ctx)
	if err != nil {
		return err
	}
	if meta.ActiveVersion == version {
		return domain.ErrSlotMapAlreadyActive
	}
	if meta.ActiveVersion > version {
		return fmt.Errorf("slot map version %d is older than active %d", version, meta.ActiveVersion)
	}
	count, err := q.CountSlotMapRowsForVersion(ctx, version)
	if err != nil {
		return err
	}
	if count == 0 {
		return domain.ErrSlotMapVersionNotFound
	}
	if count != domain.SlotCount {
		return domain.ErrSlotMapIncomplete
	}

	for _, row := range migrating {
		if err := q.UpdateSlotMapEntry(ctx, db.UpdateSlotMapEntryParams{
			Version: version,
			Slot:    row.Slot,
			ShardID: row.ShardID,
			State:   db.RedisSlotStateDRAINING,
		}); err != nil {
			return err
		}
		if err := q.UpdateSlotMigrationState(ctx, db.UpdateSlotMigrationStateParams{
			Version: version,
			Slot:    row.Slot,
			State:   db.RedisSlotMigrationStateDraining,
		}); err != nil {
			return err
		}
	}

	if err := q.SetSlotMapActiveVersion(ctx, version); err != nil {
		return err
	}

	host.AuditLog(ctx, q, adminID, "SLOT_MAP_ACTIVATED", "redis_slot_map", nil, slotMapActivatedAudit{
		Version:          version,
		MigratedSlots:    len(migrating),
		MigrationCutover: true,
	}, nil)

	if err := tx.Commit(ctx); err != nil {
		return err
	}
	host.AfterSlotMapActivated(ctx, version)
	return nil
}

func DrainMigratingSlots(ctx context.Context, host Host, version int32) error {
	if len(host.RedisShards()) == 0 {
		return fmt.Errorf("no redis shards configured")
	}
	migRepo := domain.NewSlotMigrationRepo(host.Pool())
	jobs, err := migRepo.ListDraining(ctx)
	if err != nil {
		return err
	}
	mapRepo := domain.NewSlotMapRepo(host.Pool())
	active, err := mapRepo.GetActiveVersion(ctx)
	if err != nil {
		return err
	}
	if version != 0 && version != active {
		return fmt.Errorf("drain requested for version %d but active is %d", version, active)
	}

	campaignIDs, err := host.ListActiveCampaignUUIDs(ctx)
	if err != nil {
		return err
	}
	migrator := &domain.CampaignKeyMigrator{}

	for _, job := range jobs {
		if job.Version != active {
			continue
		}
		if job.SourceShard < 0 || int(job.SourceShard) >= len(host.RedisShards()) {
			continue
		}
		src := host.RedisShards()[job.SourceShard]
		slotCampaigns := domain.FilterCampaignIDsBySlot(campaignIDs, job.Slot)
		for _, id := range slotCampaigns {
			if _, err := migrator.DrainCampaignKeys(ctx, src, id); err != nil {
				_ = migRepo.UpdateState(ctx, job.Version, job.Slot,
					db.RedisSlotMigrationStateFailed, err.Error())
				return fmt.Errorf("drain campaign %s slot %d: %w", id, job.Slot, err)
			}
		}
		if err := migRepo.UpdateState(ctx, job.Version, job.Slot,
			db.RedisSlotMigrationStateDone, ""); err != nil {
			return err
		}
		if err := mapRepoUpdateSlotState(ctx, host.Pool(), job.Version, job.Slot,
			job.TargetShard, db.RedisSlotStateACTIVE); err != nil {
			return err
		}
	}
	return nil
}

func RollbackSlotMapVersion(ctx context.Context, host Host, adminID uuid.UUID, previousVersion int32) error {
	tx, err := host.Pool().Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := db.New(tx)
	meta, err := q.LockSlotMapMeta(ctx)
	if err != nil {
		return err
	}
	if previousVersion >= meta.ActiveVersion {
		return fmt.Errorf("rollback target %d must be less than active %d", previousVersion, meta.ActiveVersion)
	}
	count, err := q.CountSlotMapRowsForVersion(ctx, previousVersion)
	if err != nil {
		return err
	}
	if count != domain.SlotCount {
		return domain.ErrSlotMapIncomplete
	}
	if err := q.SetSlotMapActiveVersion(ctx, previousVersion); err != nil {
		return err
	}
	host.AuditLog(ctx, q, adminID, "SLOT_MAP_ROLLBACK", "redis_slot_map", nil, slotMapRollbackAudit{
		FromVersion: meta.ActiveVersion,
		ToVersion:   previousVersion,
	}, nil)
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	host.AfterSlotMapActivated(ctx, previousVersion)
	return nil
}

func CatchUpDualWriteSlots(ctx context.Context, host Host, draftVersion int32) error {
	if !host.SlotMigrationDualWriteEnabled() || len(host.RedisShards()) == 0 {
		return nil
	}
	migRepo := domain.NewSlotMigrationRepo(host.Pool())
	jobs, err := migRepo.ListByVersion(ctx, draftVersion)
	if err != nil {
		return err
	}
	for _, job := range jobs {
		if job.State != db.RedisSlotMigrationStateDualWriting {
			continue
		}
		if job.SourceShard < 0 || int(job.SourceShard) >= len(host.RedisShards()) ||
			job.TargetShard < 0 || int(job.TargetShard) >= len(host.RedisShards()) {
			continue
		}
		src := host.RedisShards()[job.SourceShard]
		dst := host.RedisShards()[job.TargetShard]
		_, lag, err := domain.CatchUpSlotMigrationDeltas(ctx, src, dst, job.Version, job.Slot)
		if err != nil {
			return fmt.Errorf("catch-up slot %d: %w", job.Slot, err)
		}
		cfg := host.DualWriteConfig()
		if lag > cfg.LagThreshold {
			slotCampaigns, listErr := host.ListActiveCampaignUUIDs(ctx)
			if listErr != nil {
				return listErr
			}
			slotCampaigns = domain.FilterCampaignIDsBySlot(slotCampaigns, job.Slot)
			if host.MigrationFenceEnabled() && len(slotCampaigns) > 0 {
				if fenceErr := domain.BumpMigrationFences(ctx, host.Pool(), src, slotCampaigns); fenceErr != nil {
					return fmt.Errorf("dual-write fence fallback slot %d: %w", job.Slot, fenceErr)
				}
			}
			_ = domain.DisableSlotMigrationDualWrite(ctx, src)
			metrics.SlotMigrationCutoverBlockedTotal.WithLabelValues("lag_threshold").Inc()
		}
	}
	return nil
}

func finalizeDualWriteSlot(ctx context.Context, host Host, version int32,
	job db.RedisSlotMigration,
	campaignIDs []uuid.UUID,
) error {
	if job.SourceShard < 0 || int(job.SourceShard) >= len(host.RedisShards()) ||
		job.TargetShard < 0 || int(job.TargetShard) >= len(host.RedisShards()) {
		return fmt.Errorf("invalid shard indices source=%d target=%d", job.SourceShard, job.TargetShard)
	}
	src := host.RedisShards()[job.SourceShard]
	dst := host.RedisShards()[job.TargetShard]
	cfg := host.DualWriteConfig()

	lag, err := domain.SlotMigrationReplicationLag(ctx, src)
	if err != nil {
		return err
	}
	if lag > cfg.LagThreshold {
		slotCampaigns := domain.FilterCampaignIDsBySlot(campaignIDs, job.Slot)
		if host.MigrationFenceEnabled() && len(slotCampaigns) > 0 {
			if fenceErr := domain.BumpMigrationFences(ctx, host.Pool(), src, slotCampaigns); fenceErr != nil {
				return fmt.Errorf("dual-write fence fallback slot %d: %w", job.Slot, fenceErr)
			}
		}
		_ = domain.DisableSlotMigrationDualWrite(ctx, src)
		metrics.SlotMigrationCutoverBlockedTotal.WithLabelValues("lag_threshold").Inc()
		return fmt.Errorf("dual-write lag %d exceeds threshold %d for slot %d", lag, cfg.LagThreshold, job.Slot)
	}
	if lag > cfg.LagEpsilon {
		metrics.SlotMigrationCutoverBlockedTotal.WithLabelValues("lag_epsilon").Inc()
		return fmt.Errorf("%w: slot %d lag %d epsilon %d", ErrSlotMigrationLagNotCaughtUp, job.Slot, lag, cfg.LagEpsilon)
	}

	_, lag, err = domain.CatchUpSlotMigrationDeltas(ctx, src, dst, version, job.Slot)
	if err != nil {
		return err
	}
	if lag > cfg.LagEpsilon {
		metrics.SlotMigrationCutoverBlockedTotal.WithLabelValues("lag_epsilon").Inc()
		return fmt.Errorf("%w: slot %d lag %d epsilon %d", ErrSlotMigrationLagNotCaughtUp, job.Slot, lag, cfg.LagEpsilon)
	}
	slotCampaigns := domain.FilterCampaignIDsBySlot(campaignIDs, job.Slot)
	if len(slotCampaigns) > 0 {
		if err := domain.VerifyBudgetInvariant(ctx, host.Pool(), dst, slotCampaigns[0]); err != nil {
			metrics.SlotMigrationCutoverBlockedTotal.WithLabelValues("invariant").Inc()
			return err
		}
	}
	migRepo := domain.NewSlotMigrationRepo(host.Pool())
	if err := migRepo.UpdateState(ctx, version, job.Slot, db.RedisSlotMigrationStateCopied, ""); err != nil {
		return err
	}
	return domain.DisableSlotMigrationDualWrite(ctx, src)
}



func BumpFencesForPendingMigrations(ctx context.Context, host Host) error {
	if !host.MigrationFenceEnabled() || len(host.RedisShards()) == 0 {
		return nil
	}
	migRepo := domain.NewSlotMigrationRepo(host.Pool())
	draft, err := migRepo.GetMaxDraftVersionWithMigrating(ctx)
	if err != nil || draft <= 0 {
		return err
	}
	jobs, err := migRepo.ListByVersion(ctx, draft)
	if err != nil {
		return err
	}
	campaignIDs, err := host.ListActiveCampaignUUIDs(ctx)
	if err != nil {
		return err
	}
	for _, job := range jobs {
		if job.State == db.RedisSlotMigrationStateCopied ||
			job.State == db.RedisSlotMigrationStateDualWriting ||
			job.State == db.RedisSlotMigrationStateDraining ||
			job.State == db.RedisSlotMigrationStateDone {
			continue
		}
		if job.SourceShard < 0 || int(job.SourceShard) >= len(host.RedisShards()) {
			continue
		}
		slotCampaigns := domain.FilterCampaignIDsBySlot(campaignIDs, job.Slot)
		if len(slotCampaigns) == 0 {
			continue
		}
		src := host.RedisShards()[job.SourceShard]
		if err := domain.BumpMigrationFences(ctx, host.Pool(), src, slotCampaigns); err != nil {
			return fmt.Errorf("bump fences slot %d: %w", job.Slot, err)
		}
	}
	return nil
}

func VerifySlotMigrationR5(ctx context.Context, host Host) error {
	if len(host.RedisShards()) == 0 {
		return fmt.Errorf("no redis shards configured")
	}
	campaignIDs, err := host.ListActiveCampaignUUIDs(ctx)
	if err != nil {
		return err
	}
	if len(campaignIDs) == 0 {
		return nil
	}

	sharder := domain.NewStaticSlotSharder(len(host.RedisShards()))
	perShard := make(map[int][]uuid.UUID)
	for _, id := range campaignIDs {
		shard := sharder.GetShard(id)
		if len(perShard[shard]) < slotMigrationR5SamplePerShard {
			perShard[shard] = append(perShard[shard], id)
		}
	}

	for shard, ids := range perShard {
		if shard < 0 || shard >= len(host.RedisShards()) {
			continue
		}
		redisClient := host.RedisShards()[shard]
		snaps, err := domain.ReadBudgetInvariants(ctx, host.Pool(), redisClient, ids)
		if err != nil {
			return fmt.Errorf("r5 read shard %d: %w", shard, err)
		}
		for _, campID := range ids {
			snap, ok := snaps[campID]
			if !ok {
				return fmt.Errorf("r5 read shard %d campaign %s: not found", shard, campID)
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

func HasPendingSlotDrain(ctx context.Context, pool *pgxpool.Pool) (bool, error) {
	migRepo := domain.NewSlotMigrationRepo(pool)
	jobs, err := migRepo.ListDraining(ctx)
	if err != nil {
		return false, err
	}
	return len(jobs) > 0, nil
}

func mapRepoUpdateSlotState(ctx context.Context, pool *pgxpool.Pool, version int32, slot, shard int16, state db.RedisSlotState) error {
	if pool == nil {
		return fmt.Errorf("nil pool")
	}
	return db.New(pool).UpdateSlotMapEntry(ctx, db.UpdateSlotMapEntryParams{
		Version: version,
		Slot:    slot,
		ShardID: shard,
		State:   state,
	})
}

func verifyActivationKeysPostCopy(
	ctx context.Context,
	src, dst redis.Cmdable,
	catalog *domain.CampaignRedisKeyCatalog,
	campaignIDs []uuid.UUID,
) error {
	if catalog == nil || len(campaignIDs) == 0 {
		return nil
	}
	keys := make([]string, 0, len(campaignIDs))
	for _, id := range campaignIDs {
		keys = append(keys, catalog.ActivationRequiredKeys(id)...)
	}
	if len(keys) == 0 {
		return nil
	}

	srcPipe := src.Pipeline()
	srcCmds := make([]*redis.IntCmd, len(keys))
	for i, key := range keys {
		srcCmds[i] = srcPipe.Exists(ctx, key)
	}
	if _, err := srcPipe.Exec(ctx); err != nil {
		return fmt.Errorf("post-copy exists src pipeline: %w", err)
	}

	needDst := make([]string, 0, len(keys))
	for i, cmd := range srcCmds {
		n, err := cmd.Result()
		if err != nil {
			return fmt.Errorf("post-copy exists src %q: %w", keys[i], err)
		}
		if n > 0 {
			needDst = append(needDst, keys[i])
		}
	}
	if len(needDst) == 0 {
		return nil
	}

	dstPipe := dst.Pipeline()
	dstCmds := make([]*redis.IntCmd, len(needDst))
	for i, key := range needDst {
		dstCmds[i] = dstPipe.Exists(ctx, key)
	}
	if _, err := dstPipe.Exec(ctx); err != nil {
		return fmt.Errorf("post-copy exists dst pipeline: %w", err)
	}
	for i, cmd := range dstCmds {
		n, err := cmd.Result()
		if err != nil {
			return fmt.Errorf("post-copy exists dst %q: %w", needDst[i], err)
		}
		if n == 0 {
			return fmt.Errorf("post-copy verify: %q missing on target shard", needDst[i])
		}
	}
	return nil
}
