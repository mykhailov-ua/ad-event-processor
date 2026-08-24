package controlplane

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestValidateReportCronExpr_rejectsInvalid(t *testing.T) {
	require.Error(t, validateReportCronExpr("not-a-cron"))
	require.Error(t, validateReportCronExpr("* * *"))
	require.NoError(t, validateReportCronExpr("0 * * * *"))
}

func TestNextReportCronRun_hourly(t *testing.T) {
	after := time.Date(2026, 8, 24, 10, 15, 0, 0, time.UTC)
	next, err := nextReportCronRun("0 * * * *", after)
	require.NoError(t, err)
	require.Equal(t, 11, next.Hour())
	require.Equal(t, 0, next.Minute())
}

func TestViewsStore_PG_CRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	pool, cleanup := database.SetupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	custID := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO customers (id, name, balance, currency)
		VALUES ($1, 'Views PG', 0, 'USD')`, domain.ToUUID(custID))
	require.NoError(t, err)

	store := NewViewsStore(pool)
	created, err := store.CreateView(ctx, CreateViewRequest{
		CustomerID: custID.String(),
		Name:       "Weekly placements",
		ReportKey:  "placements",
		Spec:       json.RawMessage(`{"limit":25}`),
		IsShared:   true,
	}, "owner-1")
	require.NoError(t, err)
	require.NotEmpty(t, created.ID)

	list, err := store.ListView(ctx, custID.String())
	require.NoError(t, err)
	require.Len(t, list, 1)

	got, err := store.GetView(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, created.Name, got.Name)

	updated, err := store.UpdateView(ctx, created.ID, UpdateViewRequest{
		Name:      "Updated",
		ReportKey: "placements",
		Spec:      json.RawMessage(`{"limit":50}`),
		IsShared:  false,
	})
	require.NoError(t, err)
	require.Equal(t, "Updated", updated.Name)

	require.NoError(t, store.DeleteView(ctx, created.ID))
	_, err = store.GetView(ctx, created.ID)
	require.ErrorIs(t, err, ErrViewNotFound)
}

func TestReportSchedule_enqueueJob(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	pool, cleanup := database.SetupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	custID := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO customers (id, name, balance, currency)
		VALUES ($1, 'Schedule export', 0, 'USD')`, domain.ToUUID(custID))
	require.NoError(t, err)

	runner := NewReportJobRunner(t.TempDir(), ReportExportDeps{Pool: pool})
	created, err := insertReportSchedule(ctx, pool, CreateReportScheduleRequest{
		CustomerID: custID.String(),
		ReportKey:  "spend-velocity",
		CronExpr:   "0 * * * *",
		Spec:       json.RawMessage(`{"from_offset_days":7}`),
	})
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `UPDATE report_schedules SET next_run_at = NOW() - INTERVAL '1 minute' WHERE id = $1`, uuid.MustParse(created.ID))
	require.NoError(t, err)

	w := NewReportScheduleWorker(pool, runner)
	n, err := w.ProcessOnce(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, n)

	var jobCount int
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM report_jobs WHERE customer_id = $1`, domain.ToUUID(custID)).Scan(&jobCount))
	require.Equal(t, 1, jobCount)
}

func TestReportSchedule_createRejectsBadCron(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	pool, cleanup := database.SetupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	custID := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO customers (id, name, balance, currency)
		VALUES ($1, 'Schedule cron', 0, 'USD')`, domain.ToUUID(custID))
	require.NoError(t, err)

	_, err = insertReportSchedule(ctx, pool, CreateReportScheduleRequest{
		CustomerID: custID.String(),
		ReportKey:  "placements",
		CronExpr:   "bad cron",
	})
	require.Error(t, err)
}
