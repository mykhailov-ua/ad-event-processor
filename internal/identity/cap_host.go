package identity

import (
	"context"
	"fmt"

	"ad-event-processor/internal/licensing"
	"ad-event-processor/internal/licensingadmin"

	"github.com/jackc/pgx/v5/pgxpool"
)

type deploymentCapHost struct {
	pool *pgxpool.Pool
}

func (h deploymentCapHost) Pool() *pgxpool.Pool { return h.pool }

func (h deploymentCapHost) DeploymentLimits() (licensing.Limits, licensing.LicenseState, bool) {
	snap, err := licensing.LoadDeploymentSnapshot(context.Background(), h.pool)
	if err != nil {
		return licensing.Limits{}, licensing.StateExpired, false
	}
	return snap.Entitlements.Limits, licensing.LicenseState(snap.State), true
}

func (h deploymentCapHost) ErrValidation(msg string) error {
	return fmt.Errorf("%w: %s", ErrValidation, msg)
}

var _ licensingadmin.CapHost = deploymentCapHost{}
