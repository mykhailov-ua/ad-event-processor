package reportjob

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReportJobNotify_failedJobOneNotification_holdout(t *testing.T) {
	t.Parallel()
	userID := uuid.New().String()
	runner := NewReportJobRunner(t.TempDir(), ExportDeps{
		WriteReport: func(ctx context.Context, path string, spec ReportJobSpec) error {
			return assert.AnError
		},
	})
	jobID, err := runner.CreateJob(context.Background(), ReportJobSpec{
		CustomerID: uuid.New().String(),
		ReportKey:  "placements",
		From:       time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339),
		To:         time.Now().UTC().Format(time.RFC3339),
		Format:     "csv",
		ExportedBy: userID,
		Notify:     ReportJobNotifySpec{Channel: "in_app"},
	}, "")
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		status, ok := runner.GetJob(context.Background(), jobID)
		return ok && status.Status == JobStatusFailed
	}, time.Second, 10*time.Millisecond)

	list, err := runner.ListExportNotifications(context.Background(), userID, 10)
	require.NoError(t, err)
	require.Len(t, list.Rows, 1)
	assert.Equal(t, exportNotificationKindFailed, list.Rows[0].Kind)
	assert.Equal(t, jobID, list.Rows[0].JobID)
	assert.Equal(t, 1, list.UnreadCount)

	listAgain, err := runner.ListExportNotifications(context.Background(), userID, 10)
	require.NoError(t, err)
	require.Len(t, listAgain.Rows, 1)
}

func TestRerunJob_clonesSpec_idempotent_holdout(t *testing.T) {
	t.Parallel()
	runner := NewReportJobRunner(t.TempDir(), ExportDeps{
		WriteReport: func(ctx context.Context, path string, spec ReportJobSpec) error {
			return os.WriteFile(path, []byte("ok"), 0o640)
		},
	})
	spec := ReportJobSpec{
		CustomerID:  uuid.New().String(),
		ReportKey:   "placements",
		From:        time.Now().UTC().Add(-48 * time.Hour).Format(time.RFC3339),
		To:          time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339),
		CompareFrom: time.Now().UTC().Add(-96 * time.Hour).Format(time.RFC3339),
		CompareTo:   time.Now().UTC().Add(-72 * time.Hour).Format(time.RFC3339),
		Format:      "csv",
		RowLimit:    500,
		Notify:      ReportJobNotifySpec{Channel: "in_app"},
	}
	originalID, err := runner.CreateJob(context.Background(), spec, "orig")
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		status, ok := runner.GetJob(context.Background(), originalID)
		return ok && status.Status == JobStatusCompleted
	}, time.Second, 10*time.Millisecond)

	rerunID, err := runner.RerunJob(context.Background(), originalID, "rerun-key")
	require.NoError(t, err)
	require.NotEqual(t, originalID, rerunID)
	require.Eventually(t, func() bool {
		status, ok := runner.GetJob(context.Background(), rerunID)
		return ok && status.Status == JobStatusCompleted
	}, time.Second, 10*time.Millisecond)

	rerunAgain, err := runner.RerunJob(context.Background(), originalID, "rerun-key")
	require.NoError(t, err)
	assert.Equal(t, rerunID, rerunAgain)

	loaded, ok, err := runner.LoadJobSpec(context.Background(), rerunID)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, spec.ReportKey, loaded.ReportKey)
	assert.Equal(t, spec.CompareFrom, loaded.CompareFrom)
	assert.Equal(t, spec.CompareTo, loaded.CompareTo)
	assert.Equal(t, spec.RowLimit, loaded.RowLimit)
}

func TestValidateReportJobCompareSpec_holdout(t *testing.T) {
	t.Parallel()
	err := validateReportJobCompareSpec(ReportJobSpec{
		CompareFrom: time.Now().UTC().Add(-48 * time.Hour).Format(time.RFC3339),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "compare_from and compare_to")
}

func TestNormalizeReportJobNotify_inApp_holdout(t *testing.T) {
	t.Parallel()
	spec := ReportJobSpec{}
	require.NoError(t, normalizeReportJobNotify(&spec))
	assert.Equal(t, "none", spec.Notify.Channel)

	spec.Notify.Channel = "in_app"
	require.NoError(t, normalizeReportJobNotify(&spec))
	assert.Equal(t, "in_app", spec.Notify.Channel)
}
