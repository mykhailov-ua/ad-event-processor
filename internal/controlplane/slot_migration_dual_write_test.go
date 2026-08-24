package controlplane

import (
	"testing"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestSlotMigration_DualWriteCopyAndActivate(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}
	rdb, cleanup := database.SetupTestRedis(t)
	defer cleanup()
	rdbs := []redis.UniversalClient{rdb, rdb, rdb, rdb}
	svc, pool, ctx := setupSlotMigrationFault(t, rdbs)
	svc.cfg = &config.Config{
		SlotMigrationEnabled:          false,
		MigrationFenceEnabled:         true,
		SlotMigrationDualWriteEnabled: true,
		SlotMigrationLagEpsilon:       0,
		SlotMigrationLagThreshold:     1000,
	}

	const slot int16 = 8
	campID, _ := seedCampaignForSlot(t, svc, pool, ctx, slot, rdbs[0])
	mapRepo := domain.NewSlotMapRepo(pool)
	v := prepareMigratingVersion(t, ctx, mapRepo, slot, 2)

	require.NoError(t, svc.CopyAllMigratingSlots(ctx, v))
	migrations, err := svc.GetSlotMigrations(ctx, v)
	require.NoError(t, err)
	require.Len(t, migrations, 1)
	require.Equal(t, "dual_writing", migrations[0].State)

	flag, err := rdbs[0].Get(ctx, domain.SlotMigrationDualWriteFlagKey).Result()
	require.NoError(t, err)
	require.NotEmpty(t, flag)

	require.NoError(t, domain.PublishSlotMigrationDeltaTestHelper(ctx, rdbs[0], domain.SlotMigrationDelta{
		CampaignID: campID,
		Amount:     500,
		SpendKey:   domain.BudgetCampaignKey(campID),
	}))
	require.NoError(t, svc.CatchUpDualWriteSlots(ctx, v))

	require.NoError(t, svc.ActivateSlotMapVersion(ctx, uuid.Nil, v))
	domain.AssertBudgetInvariant(t, ctx, pool, rdbs[2], campID)
}

func TestSlotMigration_DualWriteActivateBlockedOnLag(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}
	rdb, cleanup := database.SetupTestRedis(t)
	defer cleanup()
	rdbs := []redis.UniversalClient{rdb, rdb, rdb, rdb}
	svc, pool, ctx := setupSlotMigrationFault(t, rdbs)
	svc.cfg = &config.Config{
		SlotMigrationDualWriteEnabled: true,
		SlotMigrationLagEpsilon:       0,
	}

	const slot int16 = 8
	campID, _ := seedCampaignForSlot(t, svc, pool, ctx, slot, rdbs[0])
	mapRepo := domain.NewSlotMapRepo(pool)
	v := prepareMigratingVersion(t, ctx, mapRepo, slot, 2)
	require.NoError(t, svc.CopyAllMigratingSlots(ctx, v))

	require.NoError(t, domain.PublishSlotMigrationDeltaTestHelper(ctx, rdbs[0], domain.SlotMigrationDelta{
		CampaignID: campID,
		Amount:     100,
		SpendKey:   domain.BudgetCampaignKey(campID),
	}))

	err := svc.ActivateSlotMapVersion(ctx, uuid.Nil, v)
	require.ErrorIs(t, err, ErrSlotMigrationLagNotCaughtUp)

	job, err := domain.NewSlotMigrationRepo(pool).Get(ctx, v, slot)
	require.NoError(t, err)
	require.Equal(t, db.RedisSlotMigrationStateDualWriting, job.State)
}
