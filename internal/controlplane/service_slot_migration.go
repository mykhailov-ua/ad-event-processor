package controlplane

import (
	"context"
	"errors"
	"fmt"
	"time"

	"espx/internal/domain"
	db "espx/internal/domain/db"
	"espx/internal/metrics"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const slotMigrationR5SamplePerShard = 3

type SlotMigrationOrchestrator struct {
	svc      *Service
	interval time.Duration
}

func NewSlotMigrationOrchestrator(svc *Service, interval time.Duration) *SlotMigrationOrchestrator {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	return &SlotMigrationOrchestrator{svc: svc, interval: interval}
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
	migRepo := domain.NewSlotMigrationRepo(o.svc.GetPool())
	draft, err := migRepo.GetMaxDraftVersionWithMigrating(ctx)
	if err != nil {
		if o.svc.alerter != nil {
			o.svc.alerter.AlertSlotMigrationError("draft_lookup", err)
		}
		return
	}
	if draft > 0 {
		if err := o.svc.CatchUpDualWriteSlots(ctx, draft); err != nil {
			if o.svc.alerter != nil {
				o.svc.alerter.AlertSlotMigrationError("dual_write_catchup", err)
			}
		}
		if err := o.svc.CopyAllMigratingSlots(ctx, draft); err != nil {
			if o.svc.alerter != nil {
				o.svc.alerter.AlertSlotMigrationError("copy", err)
			}
		}
	}

	mapRepo := domain.NewSlotMapRepo(o.svc.GetPool())
	active, err := mapRepo.GetActiveVersion(ctx)
	if err != nil {
		if o.svc.alerter != nil {
			o.svc.alerter.AlertSlotMigrationError("active_lookup", err)
		}
		return
	}
	if err := o.svc.DrainMigratingSlots(ctx, active); err != nil {
		if o.svc.alerter != nil {
			o.svc.alerter.AlertSlotMigrationError("drain", err)
		}
	} else {
		pending, pendErr := o.svc.HasPendingSlotDrain(ctx)
		if pendErr == nil && !pending {
			if r5Err := o.svc.VerifySlotMigrationR5(ctx); r5Err != nil && o.svc.alerter != nil {
				o.svc.alerter.AlertSlotMigrationError("r5_verify", r5Err)
			}
		}
	}
	o.svc.CheckStuckDrainJobs(ctx)
}

func (o *SlotMigrationOrchestrator) bumpPendingMigrationFences(ctx context.Context) {
	if err := o.svc.BumpFencesForPendingMigrations(ctx); err != nil && o.svc.alerter != nil {
		o.svc.alerter.AlertSlotMigrationError("bump_fences", err)
	}
}

var (
	ErrSlotMigrationNotReady       = errors.New("slot migration copy not complete for all MIGRATING slots")
	ErrSlotMigrationNoDraft        = errors.New("no draft slot map version with MIGRATING slots")
	ErrSlotMigrationKeysMissing    = errors.New("slot migration target shard missing required keys")
	ErrSlotMigrationLagNotCaughtUp = errors.New("slot migration dual-write lag above epsilon")
)

type SlotMigrationDTO struct {
	Version         int32  `json:"version"`
	Slot            int16  `json:"slot"`
	SourceShard     int16  `json:"source_shard"`
	TargetShard     int16  `json:"target_shard"`
	State           string `json:"state"`
	CampaignsTotal  int32  `json:"campaigns_total"`
	CampaignsCopied int32  `json:"campaigns_copied"`
	LastError       string `json:"last_error,omitempty"`
}

func (s *Service) GetSlotMigrations(ctx context.Context, version int32) ([]SlotMigrationDTO, error) {
	repo := domain.NewSlotMigrationRepo(s.GetPool())
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

func (s *Service) EnsureSlotMigrationJobs(ctx context.Context, draftVersion int32) error {
	mapRepo := domain.NewSlotMapRepo(s.GetPool())
	migRepo := domain.NewSlotMigrationRepo(s.GetPool())

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

func (s *Service) CopySlotMigrationData(ctx context.Context, version int32, slot int16) error {
	if len(s.rdbs) == 0 {
		return fmt.Errorf("no redis shards configured")
	}
	migRepo := domain.NewSlotMigrationRepo(s.GetPool())
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
	if job.SourceShard < 0 || int(job.SourceShard) >= len(s.rdbs) ||
		job.TargetShard < 0 || int(job.TargetShard) >= len(s.rdbs) {
		return fmt.Errorf("invalid shard indices source=%d target=%d", job.SourceShard, job.TargetShard)
	}

	campaignIDs, err := s.listActiveCampaignUUIDs(ctx)
	if err != nil {
		return err
	}
	slotCampaigns := domain.FilterCampaignIDsBySlot(campaignIDs, slot)
	total := int32(len(slotCampaigns))

	if err := migRepo.UpdateProgress(ctx, version, slot, total, job.CampaignsCopied,
		db.RedisSlotMigrationStateCopying, ""); err != nil {
		return err
	}

	src := s.rdbs[job.SourceShard]
	dst := s.rdbs[job.TargetShard]

	if s.cfg != nil && s.cfg.MigrationFenceEnabled && !s.slotMigrationDualWriteEnabled() && len(slotCampaigns) > 0 {
		if err := domain.BumpMigrationFences(ctx, s.GetPool(), src, slotCampaigns); err != nil {
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
	for _, id := range slotCampaigns {
		for _, key := range catalog.ActivationRequiredKeys(id) {
			srcExists, err := src.Exists(ctx, key).Result()
			if err != nil {
				return fmt.Errorf("post-copy exists src %q: %w", key, err)
			}
			if srcExists == 0 {
				continue
			}
			dstExists, err := dst.Exists(ctx, key).Result()
			if err != nil {
				return fmt.Errorf("post-copy exists dst %q: %w", key, err)
			}
			if dstExists == 0 {
				_ = migRepo.UpdateProgress(ctx, version, slot, total, copied,
					db.RedisSlotMigrationStateFailed, "missing key on target: "+key)
				return fmt.Errorf("post-copy verify: %q missing on target shard", key)
			}
		}
	}

	finalState := db.RedisSlotMigrationStateCopied
	if s.slotMigrationDualWriteEnabled() {
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

func (s *Service) CopyAllMigratingSlots(ctx context.Context, draftVersion int32) error {
	if err := s.EnsureSlotMigrationJobs(ctx, draftVersion); err != nil {
		return err
	}
	mapRepo := domain.NewSlotMapRepo(s.GetPool())
	migrating, err := mapRepo.ListMigratingSlots(ctx, draftVersion)
	if err != nil {
		return err
	}
	for _, row := range migrating {
		if err := s.CopySlotMigrationData(ctx, draftVersion, row.Slot); err != nil {
			return fmt.Errorf("slot migration copy version=%d slot=%d: %w", draftVersion, row.Slot, err)
		}
	}
	return nil
}

func (s *Service) ActivateSlotMapVersionWithMigration(ctx context.Context, adminID uuid.UUID, version int32) error {
	mapRepo := domain.NewSlotMapRepo(s.GetPool())
	migRepo := domain.NewSlotMigrationRepo(s.GetPool())

	migrating, err := mapRepo.ListMigratingSlots(ctx, version)
	if err != nil {
		return err
	}
	campaignIDs, err := s.listActiveCampaignUUIDs(ctx)
	if err != nil {
		return err
	}
	catalog := domain.DefaultCampaignRedisKeyCatalog

	if len(migrating) > 0 {
		if err := s.EnsureSlotMigrationJobs(ctx, version); err != nil {
			return err
		}
		for _, row := range migrating {
			job, err := migRepo.Get(ctx, version, row.Slot)
			if err != nil {
				return err
			}
			skipRewarm := false
			if job.State == db.RedisSlotMigrationStateDualWriting {
				if err := s.finalizeDualWriteSlot(ctx, version, job, campaignIDs); err != nil {
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
			if job.TargetShard < 0 || int(job.TargetShard) >= len(s.rdbs) {
				return fmt.Errorf("invalid target shard %d for slot %d", job.TargetShard, row.Slot)
			}
			slotCampaigns := domain.FilterCampaignIDsBySlot(campaignIDs, row.Slot)
			dst := s.rdbs[job.TargetShard]
			if !skipRewarm {
				if err := domain.RewarmCampaignBudgetKeys(ctx, s.GetPool(), dst, slotCampaigns); err != nil {
					return fmt.Errorf("pg re-warm slot %d: %w", row.Slot, err)
				}
			}
			if err := catalog.VerifySlotCampaignKeysExist(ctx, dst, slotCampaigns); err != nil {
				metrics.SlotMigrationCutoverBlockedTotal.WithLabelValues("missing_keys").Inc()
				return fmt.Errorf("%w: %v", ErrSlotMigrationKeysMissing, err)
			}
		}
	}

	tx, err := s.GetPool().Begin(ctx)
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

	s.AuditLog(ctx, q, adminID, "SLOT_MAP_ACTIVATED", "redis_slot_map", nil, auditSlotMapActivated{
		Version:          version,
		MigratedSlots:    len(migrating),
		MigrationCutover: true,
	}, nil)

	if err := tx.Commit(ctx); err != nil {
		return err
	}
	s.afterSlotMapActivated(ctx, version)
	return nil
}

func (s *Service) DrainMigratingSlots(ctx context.Context, version int32) error {
	if len(s.rdbs) == 0 {
		return fmt.Errorf("no redis shards configured")
	}
	migRepo := domain.NewSlotMigrationRepo(s.GetPool())
	jobs, err := migRepo.ListDraining(ctx)
	if err != nil {
		return err
	}
	mapRepo := domain.NewSlotMapRepo(s.GetPool())
	active, err := mapRepo.GetActiveVersion(ctx)
	if err != nil {
		return err
	}
	if version != 0 && version != active {
		return fmt.Errorf("drain requested for version %d but active is %d", version, active)
	}

	campaignIDs, err := s.listActiveCampaignUUIDs(ctx)
	if err != nil {
		return err
	}
	migrator := &domain.CampaignKeyMigrator{}

	for _, job := range jobs {
		if job.Version != active {
			continue
		}
		if job.SourceShard < 0 || int(job.SourceShard) >= len(s.rdbs) {
			continue
		}
		src := s.rdbs[job.SourceShard]
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
		if err := mapRepoUpdateSlotState(ctx, s.GetPool(), job.Version, job.Slot,
			job.TargetShard, db.RedisSlotStateACTIVE); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) RollbackSlotMapVersion(ctx context.Context, adminID uuid.UUID, previousVersion int32) error {
	tx, err := s.GetPool().Begin(ctx)
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
	s.AuditLog(ctx, q, adminID, "SLOT_MAP_ROLLBACK", "redis_slot_map", nil, auditSlotMapRollback{
		FromVersion: meta.ActiveVersion,
		ToVersion:   previousVersion,
	}, nil)
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	s.afterSlotMapActivated(ctx, previousVersion)
	return nil
}

func (s *Service) CatchUpDualWriteSlots(ctx context.Context, draftVersion int32) error {
	if !s.slotMigrationDualWriteEnabled() || len(s.rdbs) == 0 {
		return nil
	}
	migRepo := domain.NewSlotMigrationRepo(s.GetPool())
	jobs, err := migRepo.ListByVersion(ctx, draftVersion)
	if err != nil {
		return err
	}
	for _, job := range jobs {
		if job.State != db.RedisSlotMigrationStateDualWriting {
			continue
		}
		if job.SourceShard < 0 || int(job.SourceShard) >= len(s.rdbs) ||
			job.TargetShard < 0 || int(job.TargetShard) >= len(s.rdbs) {
			continue
		}
		src := s.rdbs[job.SourceShard]
		dst := s.rdbs[job.TargetShard]
		_, lag, err := domain.CatchUpSlotMigrationDeltas(ctx, src, dst, job.Version, job.Slot)
		if err != nil {
			return fmt.Errorf("catch-up slot %d: %w", job.Slot, err)
		}
		cfg := s.dualWriteConfig()
		if lag > cfg.LagThreshold {
			slotCampaigns, listErr := s.listActiveCampaignUUIDs(ctx)
			if listErr != nil {
				return listErr
			}
			slotCampaigns = domain.FilterCampaignIDsBySlot(slotCampaigns, job.Slot)
			if s.cfg != nil && s.cfg.MigrationFenceEnabled && len(slotCampaigns) > 0 {
				if fenceErr := domain.BumpMigrationFences(ctx, s.GetPool(), src, slotCampaigns); fenceErr != nil {
					return fmt.Errorf("dual-write fence fallback slot %d: %w", job.Slot, fenceErr)
				}
			}
			_ = domain.DisableSlotMigrationDualWrite(ctx, src)
			metrics.SlotMigrationCutoverBlockedTotal.WithLabelValues("lag_threshold").Inc()
		}
	}
	return nil
}

func (s *Service) finalizeDualWriteSlot(
	ctx context.Context,
	version int32,
	job db.RedisSlotMigration,
	campaignIDs []uuid.UUID,
) error {
	if job.SourceShard < 0 || int(job.SourceShard) >= len(s.rdbs) ||
		job.TargetShard < 0 || int(job.TargetShard) >= len(s.rdbs) {
		return fmt.Errorf("invalid shard indices source=%d target=%d", job.SourceShard, job.TargetShard)
	}
	src := s.rdbs[job.SourceShard]
	dst := s.rdbs[job.TargetShard]
	cfg := s.dualWriteConfig()

	lag, err := domain.SlotMigrationReplicationLag(ctx, src)
	if err != nil {
		return err
	}
	if lag > cfg.LagThreshold {
		slotCampaigns := domain.FilterCampaignIDsBySlot(campaignIDs, job.Slot)
		if s.cfg != nil && s.cfg.MigrationFenceEnabled && len(slotCampaigns) > 0 {
			if fenceErr := domain.BumpMigrationFences(ctx, s.GetPool(), src, slotCampaigns); fenceErr != nil {
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
		if err := domain.VerifyBudgetInvariant(ctx, s.GetPool(), dst, slotCampaigns[0]); err != nil {
			metrics.SlotMigrationCutoverBlockedTotal.WithLabelValues("invariant").Inc()
			return err
		}
	}
	migRepo := domain.NewSlotMigrationRepo(s.GetPool())
	if err := migRepo.UpdateState(ctx, version, job.Slot, db.RedisSlotMigrationStateCopied, ""); err != nil {
		return err
	}
	return domain.DisableSlotMigrationDualWrite(ctx, src)
}

func (s *Service) slotMigrationDualWriteEnabled() bool {
	return s.cfg != nil && s.cfg.SlotMigrationDualWriteEnabled
}

func (s *Service) dualWriteConfig() domain.SlotMigrationDualWriteConfig {
	cfg := domain.SlotMigrationDualWriteConfig{
		Enabled:      s.slotMigrationDualWriteEnabled(),
		LagEpsilon:   0,
		LagThreshold: 1000,
	}
	if s.cfg == nil {
		return cfg
	}
	cfg.LagEpsilon = s.cfg.SlotMigrationLagEpsilon
	cfg.LagThreshold = s.cfg.SlotMigrationLagThreshold
	if cfg.LagThreshold <= 0 {
		cfg.LagThreshold = 1000
	}
	return cfg
}

func (s *Service) BumpFencesForPendingMigrations(ctx context.Context) error {
	if s.cfg == nil || !s.cfg.MigrationFenceEnabled || len(s.rdbs) == 0 {
		return nil
	}
	migRepo := domain.NewSlotMigrationRepo(s.GetPool())
	draft, err := migRepo.GetMaxDraftVersionWithMigrating(ctx)
	if err != nil || draft <= 0 {
		return err
	}
	jobs, err := migRepo.ListByVersion(ctx, draft)
	if err != nil {
		return err
	}
	campaignIDs, err := s.listActiveCampaignUUIDs(ctx)
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
		if job.SourceShard < 0 || int(job.SourceShard) >= len(s.rdbs) {
			continue
		}
		slotCampaigns := domain.FilterCampaignIDsBySlot(campaignIDs, job.Slot)
		if len(slotCampaigns) == 0 {
			continue
		}
		src := s.rdbs[job.SourceShard]
		if err := domain.BumpMigrationFences(ctx, s.GetPool(), src, slotCampaigns); err != nil {
			return fmt.Errorf("bump fences slot %d: %w", job.Slot, err)
		}
	}
	return nil
}

func (s *Service) listActiveCampaignUUIDs(ctx context.Context) ([]uuid.UUID, error) {
	rows, err := db.New(s.GetPool()).ListCampaignIDs(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]uuid.UUID, 0, len(rows))
	for _, row := range rows {
		if !row.Valid {
			continue
		}
		out = append(out, uuid.UUID(row.Bytes))
	}
	return out, nil
}

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
