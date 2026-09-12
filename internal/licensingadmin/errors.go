package licensingadmin

import "errors"

var (
	ErrLicenseWatcherUnavailable      = errors.New("license watcher not configured")
	ErrDeploymentCampaignLimit        = errors.New("deployment active campaign limit reached for license tier")
	ErrDeploymentTenantLimit          = errors.New("deployment tenant limit reached for license tier")
	ErrDeploymentAPIKeyLimit          = errors.New("deployment api key limit reached for license tier")
	ErrDeploymentRegionLimit          = errors.New("deployment region limit reached for license tier")
	ErrDeploymentExportDisabled       = errors.New("report exports disabled for license tier")
	ErrDeploymentMonthlyEventsLimit   = errors.New("deployment monthly events limit reached for license tier")
	ErrDeploymentCostSyncNetworkLimit = errors.New("cost sync network limit reached for license tier")
)
