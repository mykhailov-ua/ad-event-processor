package billingadmin

import (
	"errors"

	"ad-event-processor/internal/campaign"
)

var (
	ErrDeploymentTenantLimit = errors.New("deployment tenant limit reached for license tier")
	ErrInvalidTimeRange      = campaign.ErrInvalidTimeRange
)
