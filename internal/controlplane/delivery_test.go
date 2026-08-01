package controlplane

import (
	"context"
	"testing"
	"time"

	"espx/internal/database"
	"espx/internal/domain"
	"espx/internal/domain/db"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testCampaignSpec(customerID uuid.UUID, name string, budgetMicro int64, idem string) CampaignCreateSpec {
	return CampaignCreateSpec{
		CustomerID:     customerID,
		Name:           name,
		BudgetLimitMicro:    budgetMicro,
		PacingMode:     string(db.PacingModeTypeASAP),
		DailyBudgetMicro:    0,
		Timezone:       "UTC",
		FreqWindow:     86400,
		DaypartHours:   []int16{},
		IdempotencyKey: idem,
	}
}

func TestResolveScheduleStatus(t *testing.T) {
	now := time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	start := now.Add(2 * time.Hour)
	end := now.Add(24 * time.Hour)

	assert.Equal(t, db.CampaignStatusTypePAUSED, resolveScheduleStatus(now, &start, &end))
	assert.Equal(t, db.CampaignStatusTypeACTIVE, resolveScheduleStatus(now.Add(3*time.Hour), &start, &end))
	assert.Equal(t, db.CampaignStatusTypePAUSED, resolveScheduleStatus(end.Add(time.Minute), &start, &end))
}

func TestCampaignTemplateCloneAndPauseResume(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	svc := NewService(pool, []redis.UniversalClient{rdb}, domain.NewJumpHashSharder(1), nil)
	defer svc.Close()

	custID := uuid.New()
	require.NoError(t, svc.CreateCustomer(context.Background(), custID, "Tpl Customer", 500_000_000, "USD"))

	tplID, err := svc.CreateCampaignTemplate(context.Background(), custID, "Base", 50_000_000,
		db.PacingModeTypeEVEN, 10_000_000, "UTC", 3, 3600, []string{"US"}, nil, []int16{9, 10, 11})
	require.NoError(t, err)

	campID, err := svc.CreateCampaignFromTemplate(context.Background(), tplID, custID, "Cloned", nil, "clone-idem")
	require.NoError(t, err)

	camp, err := svc.GetCampaignRow(context.Background(), campID)
	require.NoError(t, err)
	assert.Equal(t, db.PacingModeTypeEVEN, camp.PacingMode)
	assert.Equal(t, []string{"US"}, camp.TargetCountries)
	assert.Equal(t, []int16{9, 10, 11}, camp.DaypartHours)

	require.NoError(t, svc.PauseCampaign(context.Background(), campID, "manual"))
	camp, err = svc.GetCampaignRow(context.Background(), campID)
	require.NoError(t, err)
	assert.Equal(t, db.CampaignStatusTypePAUSED, camp.Status)

	require.NoError(t, svc.ResumeCampaign(context.Background(), campID, "manual"))
	camp, err = svc.GetCampaignRow(context.Background(), campID)
	require.NoError(t, err)
	assert.Equal(t, db.CampaignStatusTypeACTIVE, camp.Status)
}

func TestScheduledCampaignStartsPaused(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()

	svc := NewService(pool, nil, nil, nil)
	defer svc.Close()

	custID := uuid.New()
	require.NoError(t, svc.CreateCustomer(context.Background(), custID, "Sched", 200_000_000, "USD"))

	start := time.Now().Add(2 * time.Hour)
	id, err := svc.CreateCampaign(context.Background(), CampaignCreateSpec{
		CustomerID: custID, Name: "Future", BudgetLimitMicro: 50_000_000,
		PacingMode: string(db.PacingModeTypeASAP), Timezone: "UTC", FreqWindow: 86400,
		StartAt: &start, IdempotencyKey: "sched-idem",
	})
	require.NoError(t, err)

	camp, err := svc.GetCampaignRow(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, db.CampaignStatusTypePAUSED, camp.Status)
}

func TestValidateDaypartHours(t *testing.T) {
	assert.NoError(t, validateDaypartHours([]int16{0, 23}))
	assert.Error(t, validateDaypartHours([]int16{24}))
}
