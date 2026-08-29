package export

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ad-event-processor/internal/reports"

	"ad-event-processor/internal/reportjob"

	"github.com/google/uuid"
)

func TestWriteReportCSV_supportsAllLiveReportKeys(t *testing.T) {
	deps := reports.ReportExportDeps{}
	from := time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339)
	to := time.Now().UTC().Format(time.RFC3339)
	for _, key := range reports.LiveReportExportKeys() {
		key := key
		t.Run(key, func(t *testing.T) {
			err := writeReportCSV(context.Background(), deps, filepath.Join(t.TempDir(), key+".csv"), reportjob.ReportJobSpec{
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
