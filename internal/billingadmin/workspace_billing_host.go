package billingadmin

import (
	"ad-event-processor/internal/licensing"

	"github.com/jackc/pgx/v5/pgxpool"
)

type WorkspaceBillingHost interface {
	Pool() *pgxpool.Pool
	MapCustomerNotFound(err error) error
	ExportChunkMaxBytes() int
	DeploymentLimits() (licensing.Limits, licensing.LicenseState, bool)
	ErrValidation(msg string) error
}

type WorkspaceBilling struct {
	host WorkspaceBillingHost
}

func NewWorkspaceBilling(host WorkspaceBillingHost) *WorkspaceBilling {
	return &WorkspaceBilling{host: host}
}
