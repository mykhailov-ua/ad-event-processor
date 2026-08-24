package controlplane

import (
	"context"
	"encoding/json"
	"fmt"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
)

type SlotMapDTO struct {
	Slot    int16  `json:"slot"`
	ShardID int16  `json:"shard_id"`
	State   string `json:"state"`
}

type SlotMapVersionDTO struct {
	Version        int32        `json:"version"`
	ActiveVersion  int32        `json:"active_version"`
	SlotCount      int32        `json:"slot_count"`
	MigratingCount int          `json:"migrating_count"`
	Slots          []SlotMapDTO `json:"slots,omitempty"`
}

func (s *Service) GetSlotMap(ctx context.Context, version *int32, includeSlots bool) (SlotMapVersionDTO, error) {
	repo := domain.NewSlotMapRepo(s.GetPool())
	active, err := repo.GetActiveVersion(ctx)
	if err != nil {
		return SlotMapVersionDTO{}, err
	}

	target := active
	if version != nil {
		target = *version
	}

	rows, err := repo.ListVersion(ctx, target)
	if err != nil {
		return SlotMapVersionDTO{}, err
	}

	dto := SlotMapVersionDTO{
		Version:       target,
		ActiveVersion: active,
		SlotCount:     int32(len(rows)),
	}
	migrating, err := repo.ListMigratingSlots(ctx, target)
	if err != nil {
		return SlotMapVersionDTO{}, err
	}
	dto.MigratingCount = len(migrating)

	if includeSlots {
		dto.Slots = make([]SlotMapDTO, 0, len(rows))
		for _, row := range rows {
			dto.Slots = append(dto.Slots, SlotMapDTO{
				Slot:    row.Slot,
				ShardID: row.ShardID,
				State:   string(row.State),
			})
		}
	}
	return dto, nil
}

func (s *Service) CreateSlotMapVersion(ctx context.Context, adminID uuid.UUID, baseVersion *int32, overrides []domain.SlotOverride) (int32, error) {
	base := int32(0)
	if baseVersion != nil {
		base = *baseVersion
	} else {
		active, err := domain.NewSlotMapRepo(s.GetPool()).GetActiveVersion(ctx)
		if err != nil {
			return 0, err
		}
		base = active
	}

	for _, o := range overrides {
		if o.Slot < 0 || o.Slot > domain.SlotMask || o.ShardID < 0 {
			return 0, fmt.Errorf("invalid slot override: slot=%d shard=%d", o.Slot, o.ShardID)
		}
	}

	tx, err := s.GetPool().Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := db.New(tx)
	if _, err := q.LockSlotMapMeta(ctx); err != nil {
		return 0, err
	}

	baseCount, err := q.CountSlotMapRowsForVersion(ctx, base)
	if err != nil {
		return 0, err
	}
	if baseCount == 0 {
		return 0, domain.ErrSlotMapVersionNotFound
	}
	if baseCount != domain.SlotCount {
		return 0, domain.ErrSlotMapIncomplete
	}

	maxVersion, err := q.GetMaxSlotMapVersion(ctx)
	if err != nil {
		return 0, err
	}
	newVersion := maxVersion + 1

	if err := q.CopySlotMapVersion(ctx, db.CopySlotMapVersionParams{
		Version:   base,
		Version_2: newVersion,
	}); err != nil {
		return 0, err
	}
	for _, o := range overrides {
		state := o.State
		if state == "" {
			state = db.RedisSlotStateACTIVE
		}
		if err := q.UpdateSlotMapEntry(ctx, db.UpdateSlotMapEntryParams{
			Version: newVersion,
			Slot:    o.Slot,
			ShardID: o.ShardID,
			State:   state,
		}); err != nil {
			return 0, err
		}
	}
	newCount, err := q.CountSlotMapRowsForVersion(ctx, newVersion)
	if err != nil {
		return 0, err
	}
	if newCount != domain.SlotCount {
		return 0, domain.ErrSlotMapIncomplete
	}

	overridesJSON, err := json.Marshal(overrides)
	if err != nil {
		return 0, err
	}
	s.AuditLog(ctx, q, adminID, "SLOT_MAP_VERSION_CREATED", "redis_slot_map", nil, auditSlotMapVersionCreated{
		BaseVersion: base,
		NewVersion:  newVersion,
		Overrides:   overridesJSON,
	}, nil)

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return newVersion, nil
}

func (s *Service) MarkSlotMapMigrating(ctx context.Context, adminID uuid.UUID, version int32, slots []int16, targetShard int16) error {
	if targetShard < 0 {
		return domain.ErrSlotMapInvalidShard
	}

	tx, err := s.GetPool().Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := db.New(tx)
	versionCount, err := q.CountSlotMapRowsForVersion(ctx, version)
	if err != nil {
		return err
	}
	if versionCount == 0 {
		return domain.ErrSlotMapVersionNotFound
	}

	for _, slot := range slots {
		if slot < 0 || slot > domain.SlotMask {
			return domain.ErrSlotMapInvalidSlot
		}
		if _, err := q.LockSlotMapEntry(ctx, db.LockSlotMapEntryParams{
			Version: version,
			Slot:    slot,
		}); err != nil {
			return err
		}
		if err := q.UpdateSlotMapEntry(ctx, db.UpdateSlotMapEntryParams{
			Version: version,
			Slot:    slot,
			ShardID: targetShard,
			State:   db.RedisSlotStateMIGRATING,
		}); err != nil {
			return err
		}
	}

	slotsJSON, err := json.Marshal(slots)
	if err != nil {
		return err
	}
	s.AuditLog(ctx, q, adminID, "SLOT_MAP_MARK_MIGRATING", "redis_slot_map", nil, auditSlotMapMarkMigrating{
		Version:     version,
		Slots:       slotsJSON,
		TargetShard: targetShard,
	}, nil)

	if err := tx.Commit(ctx); err != nil {
		return err
	}
	if s.alerter != nil {
		s.alerter.AlertSlotMapMigrating(ctx, version, slots, targetShard)
	}
	return nil
}

func (s *Service) ActivateSlotMapVersion(ctx context.Context, adminID uuid.UUID, version int32) error {
	return s.ActivateSlotMapVersionWithMigration(ctx, adminID, version)
}

func (s *Service) afterSlotMapActivated(ctx context.Context, version int32) {
	routingEpoch := int64(0)
	if row, err := domain.NewCampaignRoutingRepo(s.GetPool()).BumpGlobalRoutingEpoch(ctx); err == nil {
		routingEpoch = row.RoutingEpoch
		version = row.ActiveVersion
	}
	if ss, ok := s.sharder.(*domain.StaticSlotSharder); ok {
		_, _ = domain.LoadActiveSlotMap(ctx, s.GetPool(), ss, len(s.rdbs))
	}
	s.publishRoutingCutover(ctx, routingEpoch, version)
}
