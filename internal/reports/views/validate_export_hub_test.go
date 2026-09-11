package views

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func init() {
	SetLiveReportExportKeys(func() []string {
		return []string{"placements", "campaign-toggle-cohort", "layer-desync-drilldown"}
	})
}

func TestValidateSavedViewInput_exportHubSpecKeys(t *testing.T) {
	t.Parallel()
	spec := json.RawMessage(`{
		"kind":"report",
		"entry":"report-placements",
		"from":"2026-03-01T00:00:00Z",
		"to":"2026-03-05T00:00:00Z",
		"compare_from":"2026-02-25T00:00:00Z",
		"compare_to":"2026-03-01T00:00:00Z",
		"format":"csv",
		"destination":"google_sheet",
		"row_limit":1000,
		"notify":{"channel":"in_app"},
		"google_sheet":{"mode":"create","sheet_title":"Exports"},
		"import_payload":{"layer_desync_count":2}
	}`)
	err := validateSavedViewInput("preset", "placements", spec)
	require.NoError(t, err)
}

func TestValidateSavedViewInput_billingPresetReportKey(t *testing.T) {
	t.Parallel()
	spec := json.RawMessage(`{"kind":"billing","billing_format":"ndjson"}`)
	err := validateSavedViewInput("billing preset", "billing-export", spec)
	require.NoError(t, err)
}
