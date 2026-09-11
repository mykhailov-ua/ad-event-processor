package reportjob

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildReportJobSpecFromSchedule_googleSheet_holdout(t *testing.T) {
	t.Parallel()
	ownerID := uuid.New().String()
	specJSON, err := json.Marshal(reportScheduleSpec{
		FromOffsetDays: 3,
		GoogleSheet:    ReportJobGoogleSheetSpec{Mode: "create", SheetTitle: "Daily"},
		Notify:         ReportJobNotifySpec{Channel: "in_app"},
	})
	require.NoError(t, err)
	row := reportScheduleRow{
		id:          uuid.New(),
		customerID:  uuid.New(),
		reportKey:   "pacing-drift",
		format:      "csv",
		destination: "google_sheet",
		ownerUserID: ownerID,
		specJSON:    specJSON,
		nextRunAt:   time.Now().UTC(),
	}
	spec, idem, err := buildReportJobSpecFromSchedule(row)
	require.NoError(t, err)
	assert.Contains(t, idem, "schedule:")
	assert.Equal(t, "google_sheet", spec.Destination)
	assert.Equal(t, ownerID, spec.ExportedBy)
	assert.Equal(t, "in_app", spec.Notify.Channel)
	assert.Equal(t, "create", spec.GoogleSheet.Mode)
}
