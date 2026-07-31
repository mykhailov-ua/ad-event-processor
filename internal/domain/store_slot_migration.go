package domain

import (
	"context"
	"fmt"

	"espx/internal/domain/db"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SlotMigrationRepo struct {
	pool *pgxpool.Pool
}

func NewSlotMigrationRepo(pool *pgxpool.Pool) *SlotMigrationRepo {
	return &SlotMigrationRepo{pool: pool}
}

func (r *SlotMigrationRepo) InsertIfAbsent(
	ctx context.Context,
	version int32,
	slot, sourceShard, targetShard int16,
) error {
	if r.pool == nil {
		return fmt.Errorf("slot migration repo: nil pool")
	}
	return db.New(r.pool).InsertSlotMigrationIfAbsent(ctx, db.InsertSlotMigrationIfAbsentParams{
		Version:     version,
		Slot:        slot,
		SourceShard: sourceShard,
		TargetShard: targetShard,
	})
}

func (r *SlotMigrationRepo) Upsert(
	ctx context.Context,
	version int32,
	slot, sourceShard, targetShard int16,
	state db.RedisSlotMigrationState,
	total, copied int32,
	lastErr string,
) error {
	if r.pool == nil {
		return fmt.Errorf("slot migration repo: nil pool")
	}
	return db.New(r.pool).UpsertSlotMigration(ctx, db.UpsertSlotMigrationParams{
		Version:         version,
		Slot:            slot,
		SourceShard:     sourceShard,
		TargetShard:     targetShard,
		State:           state,
		CampaignsTotal:  total,
		CampaignsCopied: copied,
		LastError:       pgText(lastErr),
	})
}

func (r *SlotMigrationRepo) Get(ctx context.Context, version int32, slot int16) (db.RedisSlotMigration, error) {
	if r.pool == nil {
		return db.RedisSlotMigration{}, fmt.Errorf("slot migration repo: nil pool")
	}
	return db.New(r.pool).GetSlotMigration(ctx, db.GetSlotMigrationParams{
		Version: version,
		Slot:    slot,
	})
}

func (r *SlotMigrationRepo) ListByVersion(ctx context.Context, version int32) ([]db.RedisSlotMigration, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("slot migration repo: nil pool")
	}
	return db.New(r.pool).ListSlotMigrationsByVersion(ctx, version)
}

func (r *SlotMigrationRepo) UpdateProgress(
	ctx context.Context,
	version int32,
	slot int16,
	total, copied int32,
	state db.RedisSlotMigrationState,
	lastErr string,
) error {
	if r.pool == nil {
		return fmt.Errorf("slot migration repo: nil pool")
	}
	return db.New(r.pool).UpdateSlotMigrationProgress(ctx, db.UpdateSlotMigrationProgressParams{
		Version:         version,
		Slot:            slot,
		CampaignsTotal:  total,
		CampaignsCopied: copied,
		State:           state,
		LastError:       pgText(lastErr),
	})
}

func (r *SlotMigrationRepo) UpdateState(
	ctx context.Context,
	version int32,
	slot int16,
	state db.RedisSlotMigrationState,
	lastErr string,
) error {
	if r.pool == nil {
		return fmt.Errorf("slot migration repo: nil pool")
	}
	return db.New(r.pool).UpdateSlotMigrationState(ctx, db.UpdateSlotMigrationStateParams{
		Version:   version,
		Slot:      slot,
		State:     state,
		LastError: pgText(lastErr),
	})
}

func pgText(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: s, Valid: true}
}

func (r *SlotMigrationRepo) ListDraining(ctx context.Context) ([]db.RedisSlotMigration, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("slot migration repo: nil pool")
	}
	return db.New(r.pool).ListDrainingSlotMigrations(ctx)
}

func (r *SlotMigrationRepo) ListByStates(ctx context.Context, states []db.RedisSlotMigrationState) ([]db.RedisSlotMigration, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("slot migration repo: nil pool")
	}
	return db.New(r.pool).ListSlotMigrationsByState(ctx, states)
}

func (r *SlotMigrationRepo) GetMaxDraftVersionWithMigrating(ctx context.Context) (int32, error) {
	if r.pool == nil {
		return 0, fmt.Errorf("slot migration repo: nil pool")
	}
	return db.New(r.pool).GetMaxDraftVersionWithMigrating(ctx)
}
