package reportjob

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReportJobRunTimeout_csvDefault2Min_holdout(t *testing.T) {
	t.Setenv("REPORT_JOB_RUN_TIMEOUT_SEC", "")

	spec := ReportJobSpec{Format: "csv", Destination: "download"}
	got := reportJobRunTimeoutForSpec(spec)
	require.Equal(t, 2*time.Minute, got)

	var (
		mu       sync.Mutex
		deadline time.Time
	)
	runner := NewReportJobRunner(t.TempDir(), ExportDeps{
		WriteReport: func(ctx context.Context, path string, spec ReportJobSpec) error {
			if dl, ok := ctx.Deadline(); ok {
				mu.Lock()
				deadline = dl
				mu.Unlock()
			}
			return os.WriteFile(path, []byte("x"), 0o600)
		},
	})
	jobID, err := runner.CreateJob(context.Background(), ReportJobSpec{
		CustomerID: uuid.New().String(),
		ReportKey:  "placements",
		From:       time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339),
		To:         time.Now().UTC().Format(time.RFC3339),
		Format:     "csv",
	}, "")
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return !deadline.IsZero()
	}, time.Second, 10*time.Millisecond)

	mu.Lock()
	gotDeadline := deadline
	mu.Unlock()
	assert.InDelta(t, 120.0, time.Until(gotDeadline).Seconds(), 5.0)

	status, ok := runner.GetJob(context.Background(), jobID)
	require.True(t, ok)
	assert.Equal(t, JobStatusCompleted, status.Status)
}

func TestReportJobRunTimeout_extendedWhenEnvSet_holdout(t *testing.T) {
	t.Setenv("REPORT_JOB_RUN_TIMEOUT_SEC", "300")

	cases := []struct {
		name string
		spec ReportJobSpec
	}{
		{
			name: "xlsx_download",
			spec: ReportJobSpec{Format: "xlsx", Destination: "download"},
		},
		{
			name: "csv_google_sheet",
			spec: ReportJobSpec{Format: "csv", Destination: "google_sheet"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := reportJobRunTimeoutForSpec(tc.spec)
			require.Equal(t, 5*time.Minute, got)
		})
	}

	var (
		mu       sync.Mutex
		deadline time.Time
	)
	runner := NewReportJobRunner(t.TempDir(), ExportDeps{
		WriteReport: func(ctx context.Context, path string, spec ReportJobSpec) error {
			if dl, ok := ctx.Deadline(); ok {
				mu.Lock()
				deadline = dl
				mu.Unlock()
			}
			return os.WriteFile(path, []byte("x"), 0o600)
		},
	})
	jobID, err := runner.CreateJob(context.Background(), ReportJobSpec{
		CustomerID: uuid.New().String(),
		ReportKey:  "placements",
		From:       time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339),
		To:         time.Now().UTC().Format(time.RFC3339),
		Format:     "xlsx",
	}, "")
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return !deadline.IsZero()
	}, time.Second, 10*time.Millisecond)

	mu.Lock()
	gotDeadline := deadline
	mu.Unlock()
	assert.InDelta(t, 300.0, time.Until(gotDeadline).Seconds(), 5.0)

	status, ok := runner.GetJob(context.Background(), jobID)
	require.True(t, ok)
	assert.Equal(t, JobStatusCompleted, status.Status)
}

func TestReportJobRunTimeout_xlsxDefault10MinWithoutEnv_holdout(t *testing.T) {
	t.Setenv("REPORT_JOB_RUN_TIMEOUT_SEC", "")

	spec := ReportJobSpec{Format: "xlsx", Destination: "download"}
	got := reportJobRunTimeoutForSpec(spec)
	require.Equal(t, 10*time.Minute, got)
}

func TestReportJobRunTimeout_failJobSanitized_holdout(t *testing.T) {
	t.Setenv("REPORT_JOB_RUN_TIMEOUT_SEC", "1")

	runner := NewReportJobRunner(t.TempDir(), ExportDeps{
		WriteReport: func(ctx context.Context, path string, spec ReportJobSpec) error {
			<-ctx.Done()
			return ctx.Err()
		},
	})
	jobID, err := runner.CreateJob(context.Background(), ReportJobSpec{
		CustomerID: uuid.New().String(),
		ReportKey:  "placements",
		From:       time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339),
		To:         time.Now().UTC().Format(time.RFC3339),
		Format:     "xlsx",
	}, "")
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		status, ok := runner.GetJob(context.Background(), jobID)
		return ok && status.Status == JobStatusFailed && status.Error != ""
	}, 5*time.Second, 25*time.Millisecond)

	status, ok := runner.GetJob(context.Background(), jobID)
	require.True(t, ok)
	assert.Equal(t, JobStatusFailed, status.Status)
	assert.Equal(t, exportErrTimeoutPublic, status.Error)
}
