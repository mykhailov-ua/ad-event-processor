package reports

import "ad-event-processor/internal/reports/views"

func init() {
	views.SetLiveReportExportKeys(LiveReportExportKeys)
}
