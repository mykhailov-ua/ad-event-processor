package controlplane

import (
	"ad-event-processor/internal/licensing"
	"ad-event-processor/internal/licensing/entitlements"
	"ad-event-processor/internal/licensingadmin"

	"github.com/jackc/pgx/v5/pgxpool"
)

type regionCapHost struct {
	pool *pgxpool.Pool
	snap entitlements.DeploymentSnapshot
}

func (h regionCapHost) Pool() *pgxpool.Pool { return h.pool }

func (h regionCapHost) DeploymentLimits() (licensing.Limits, licensing.LicenseState, bool) {
	return h.snap.Entitlements.Limits, licensing.LicenseState(h.snap.State), true
}

func (h regionCapHost) ErrValidation(msg string) error { return errValidation(msg) }

var _ licensingadmin.CapHost = regionCapHost{}
