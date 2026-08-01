package controlplane

import (
	"context"
	"time"

	"espx/internal/controlplane/adminapi"
	billingdb "espx/internal/ledger/db"
	"espx/internal/licensing"

	"github.com/jackc/pgx/v5"
)

func (h *Handler) metaEnricher() adminapi.MetaEnricher {
	return func(ctx context.Context) (string, *adminapi.MetaLicenseDTO, error) {
		if h == nil || h.svc == nil {
			return "", nil, nil
		}
		var deploymentID string
		if fbMeta, err := h.svc.SupportFeedbackMeta(ctx); err != nil {
			return "", nil, err
		} else {
			deploymentID = fbMeta.DeploymentID
		}
		pool := h.svc.GetPool()
		if pool == nil {
			return deploymentID, nil, nil
		}
		licRow, err := billingdb.New(pool).GetLicenseStatus(ctx)
		if err == pgx.ErrNoRows {
			return deploymentID, nil, nil
		}
		if err != nil {
			return "", nil, err
		}
		var validUntil time.Time
		hasValidUntil := licRow.ValidUntil.Valid
		if hasValidUntil {
			validUntil = licRow.ValidUntil.Time
		}
		license := adminapi.BuildMetaLicense(
			licRow.State,
			licensing.BannerSeverity(licensing.LicenseState(licRow.State)),
			validUntil,
			hasValidUntil,
		)
		return deploymentID, license, nil
	}
}
