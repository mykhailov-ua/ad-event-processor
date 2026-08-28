package licensingadmin

import "errors"

var (
	ErrLicenseWatcherUnavailable = errors.New("license watcher not configured")
	ErrDeploymentCampaignLimit   = errors.New("deployment active campaign limit reached for license tier")
)
