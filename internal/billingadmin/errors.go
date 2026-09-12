package billingadmin

import (
	"ad-event-processor/internal/campaign"
	"ad-event-processor/internal/licensingadmin"
)

var (
	ErrDeploymentTenantLimit          = licensingadmin.ErrDeploymentTenantLimit
	ErrDeploymentAPIKeyLimit          = licensingadmin.ErrDeploymentAPIKeyLimit
	ErrDeploymentRegionLimit          = licensingadmin.ErrDeploymentRegionLimit
	ErrDeploymentExportDisabled       = licensingadmin.ErrDeploymentExportDisabled
	ErrDeploymentMonthlyEventsLimit   = licensingadmin.ErrDeploymentMonthlyEventsLimit
	ErrDeploymentCostSyncNetworkLimit = licensingadmin.ErrDeploymentCostSyncNetworkLimit
	ErrInvalidTimeRange               = campaign.ErrInvalidTimeRange
)
