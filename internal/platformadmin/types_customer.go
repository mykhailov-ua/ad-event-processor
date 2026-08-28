package platformadmin

import (
	"ad-event-processor/internal/billingadmin"
	"ad-event-processor/internal/campaign"

	"github.com/google/uuid"
)

type (
	CustomerBalanceDTO = campaign.CustomerBalanceDTO
	BalanceLedgerDTO   = campaign.BalanceLedgerDTO
	LedgerExportResult = campaign.LedgerExportResult
	AuditLogDTO        = campaign.AuditLogDTO
	DisputeRowDTO      = billingadmin.DisputeRowDTO
	DisputeListResult  = billingadmin.DisputeListResult
)

type SupportFeedbackMeta struct {
	DeploymentID  string `json:"deployment_id"`
	BinaryVersion string `json:"binary_version"`
}

type SupportFeedbackRecord struct {
	Type          string
	ContactEmail  string
	Message       string
	AttachBundle  bool
	BundleGzip    []byte
	SubmitterID   uuid.UUID
	DeploymentID  string
	BinaryVersion string
}
