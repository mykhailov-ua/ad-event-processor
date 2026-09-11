package export

import (
	"context"
	"testing"

	"ad-event-processor/internal/integrations/googlesheets"
	"ad-event-processor/internal/reportjob"
	"ad-event-processor/internal/reports"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestWriteGoogleSheet_requiresClient_holdout(t *testing.T) {
	t.Parallel()
	_, err := WriteGoogleSheet(context.Background(), reports.ReportExportDeps{}, nil, reportjob.ReportJobSpec{
		ExportedBy: uuid.New().String(),
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "not configured")
}

func TestWriteGoogleSheet_requiresOperatorUUID_holdout(t *testing.T) {
	t.Parallel()
	client := googlesheets.NewClient(googlesheets.Config{}, nil)
	_, err := WriteGoogleSheet(context.Background(), reports.ReportExportDeps{}, client, reportjob.ReportJobSpec{
		ExportedBy: "not-a-uuid",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "connected operator")
}
