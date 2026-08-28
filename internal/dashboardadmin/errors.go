package dashboardadmin

import (
	"errors"

	"ad-event-processor/internal/reports"
)

type (
	DataFreshnessDTO         = reports.DataFreshnessDTO
	DashboardSeriesPointDTO  = reports.DashboardSeriesPointDTO
	CustomerFraudOverviewDTO = reports.CustomerFraudOverviewDTO
)

var ErrPublisherScopeRequired = errors.New("publisher seller_id bind required")

type QueryError string

func (e QueryError) Error() string { return string(e) }

func InvalidQuery(msg string) error { return QueryError(msg) }
