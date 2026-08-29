package shardadmin

import (
	"context"
	"time"

	"ad-event-processor/internal/domain"
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
