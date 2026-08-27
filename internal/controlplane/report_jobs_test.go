package controlplane

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestWriteReportCSV_supportsAllLiveReportKeys(t *testing.T) {
	runner := NewReportJobRunner(t.TempDir(), ReportExportDeps{})
	from := time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339)
	to := time.Now().UTC().Format(time.RFC3339)
	for _, key := range liveReportExportKeys() {
		key := key
		t.Run(key, func(t *testing.T) {
			err := runner.writeReportCSV(context.Background(), filepath.Join(t.TempDir(), key+".csv"), ReportJobSpec{
				CustomerID: uuid.New().String(),
				ReportKey:  key,
				From:       from,
				To:         to,
				Format:     "csv",
			})
			if err != nil && strings.Contains(err.Error(), "unsupported report_key") {
				t.Fatalf("unsupported export key %q: %v", key, err)
			}
		})
	}
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

	runner := NewReportJobRunner(t.TempDir(), ReportExportDeps{Pool: pool})
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

	runner := NewReportJobRunner(t.TempDir(), ReportExportDeps{Pool: pool})
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
