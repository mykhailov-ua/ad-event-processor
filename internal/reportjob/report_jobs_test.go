package reportjob

import (
	"context"
	"testing"
	"time"

	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReportJobRunner_cancelPending_holdout(t *testing.T) {
	t.Parallel()
	runner := NewReportJobRunner(t.TempDir(), ExportDeps{})
	customerID := uuid.New().String()
	jobID, err := runner.CreateJob(context.Background(), ReportJobSpec{
		CustomerID: customerID,
		ReportKey:  "placements",
		From:       time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339),
		To:         time.Now().UTC().Format(time.RFC3339),
	}, "")
	require.NoError(t, err)
	status, ok, err := runner.CancelJob(context.Background(), jobID)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, JobStatusCancelled, status.Status)
	got, found := runner.GetJob(context.Background(), jobID)
	require.True(t, found)
	assert.Equal(t, JobStatusCancelled, got.Status)
}

func TestReportJob_idempotencyReplay(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()

	ctx := context.Background()
	custID := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO customers (id, name, balance, currency)
		VALUES ($1, 'Report export', 0, 'USD')`, domain.ToUUID(custID))
	require.NoError(t, err)

	runner := NewReportJobRunner(t.TempDir(), ExportDeps{Pool: pool})
	from := time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339)
	to := time.Now().UTC().Format(time.RFC3339)
	spec := ReportJobSpec{
		CustomerID: custID.String(),
		ReportKey:  "spend-velocity",
		From:       from,
		To:         to,
		Format:     "csv",
	}

	job1, err := runner.CreateJob(ctx, spec, "report-export-idem-held-out")
	require.NoError(t, err)
	job2, err := runner.CreateJob(ctx, spec, "report-export-idem-held-out")
	require.NoError(t, err)
	require.Equal(t, job1, job2)
}

func TestReportJob_ProcessOnce_marksFailedWithoutCH(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()

	ctx := context.Background()
	custID := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO customers (id, name, balance, currency)
		VALUES ($1, 'Report export fault', 0, 'USD')`, domain.ToUUID(custID))
	require.NoError(t, err)

	runner := NewReportJobRunner(t.TempDir(), ExportDeps{Pool: pool})
	from := time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339)
	to := time.Now().UTC().Format(time.RFC3339)
	jobID, err := runner.CreateJob(ctx, ReportJobSpec{
		CustomerID: custID.String(),
		ReportKey:  "spend-velocity",
		From:       from,
		To:         to,
		Format:     "csv",
	}, "")
	require.NoError(t, err)

	n, err := runner.ProcessOnce(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, n)

	status, ok := runner.GetJob(ctx, jobID)
	require.True(t, ok)
	require.Equal(t, JobStatusFailed, status.Status)
	require.NotEmpty(t, status.Error)
}
