package reportjob

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildReportJobSpecFromSavedView_compareAndNotify(t *testing.T) {
	t.Parallel()
	specJSON := json.RawMessage(`{
		"from":"2026-03-01T00:00:00Z",
		"to":"2026-03-05T00:00:00Z",
		"compare_from":"2026-02-25T00:00:00Z",
		"compare_to":"2026-03-01T00:00:00Z",
		"format":"xlsx",
		"destination":"download",
		"row_limit":5000,
		"notify":{"channel":"in_app"}
	}`)
	spec, err := BuildReportJobSpecFromSavedView("11111111-1111-4111-8111-111111111111", "placements", "22222222-2222-4222-8222-222222222222", specJSON)
	require.NoError(t, err)
	require.Equal(t, "placements", spec.ReportKey)
	require.Equal(t, "xlsx", spec.Format)
	require.Equal(t, 5000, spec.RowLimit)
	require.Equal(t, "2026-02-25T00:00:00Z", spec.CompareFrom)
	require.Equal(t, "in_app", spec.Notify.Channel)
}

func TestBuildReportJobSpecFromSavedView_billingPresetRejected_holdout(t *testing.T) {
	t.Parallel()
	_, err := BuildReportJobSpecFromSavedView("11111111-1111-4111-8111-111111111111", SavedViewReportKeyBilling, "owner", json.RawMessage(`{"kind":"billing"}`))
	require.Error(t, err)
}

func TestBuildReportJobSpecFromSavedView_auditKindRejected_holdout(t *testing.T) {
	t.Parallel()
	_, err := BuildReportJobSpecFromSavedView("11111111-1111-4111-8111-111111111111", "placements", "owner", json.RawMessage(`{"kind":"audit"}`))
	require.Error(t, err)
}
