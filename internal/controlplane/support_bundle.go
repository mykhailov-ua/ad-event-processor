package controlplane

import (
	"context"
	"io"

	"espx/pkg/supportbundle"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type supportBundleWriter struct {
	pool   *pgxpool.Pool
	logDir string
}

func (w supportBundleWriter) WriteSupportBundle(ctx context.Context, out io.Writer) error {
	meta := supportbundle.Meta{}
	if w.pool != nil {
		var dep uuid.UUID
		var state string
		err := w.pool.QueryRow(ctx, `
			SELECT deployment_id, state
			FROM billing.license_status
			LIMIT 1`).Scan(&dep, &state)
		if err != nil && err != pgx.ErrNoRows {
			return err
		}
		if err == nil {
			if dep != uuid.Nil {
				meta.DeploymentID = dep.String()
			}
			meta.LicenseState = state
		}
	}
	return supportbundle.Write(ctx, out, supportbundle.Options{
		Meta:     meta,
		LogDir:   w.logDir,
		MaxBytes: supportbundle.DefaultMaxBytes,
	})
}
