package reportjob

const (
	// PG report_jobs.status CHECK values; worker claim flips PENDING->RUNNING, export ends COMPLETED/FAILED.
	JobStatusPending   = "PENDING"
	JobStatusRunning   = "RUNNING"
	JobStatusCompleted = "COMPLETED"
	JobStatusFailed    = "FAILED"
	JobStatusCancelled = "CANCELLED"

	CampaignImportValidationReportKey = "campaign-import-validation"
)
