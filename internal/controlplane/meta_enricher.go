package controlplane

import (
	"context"
	"time"

	"espx/internal/controlplane/adminapi"
	billingdb "espx/internal/ledger/db"
	"espx/internal/licensing"
	"espx/pkg/legal"

	"github.com/jackc/pgx/v5"
)

func (h *Handler) metaEnricher() adminapi.MetaEnricher {
	return func(ctx context.Context) (adminapi.MetaEnrichOut, error) {
		out := adminapi.MetaEnrichOut{}
		if h == nil || h.svc == nil {
			return out, nil
		}
		if fbMeta, err := h.svc.SupportFeedbackMeta(ctx); err != nil {
			return out, err
		} else {
			out.DeploymentID = fbMeta.DeploymentID
		}
		_, bootstrapped, err := h.svc.GetPlatformConfig(ctx)
		if err != nil {
			return out, err
		}
		out.BootstrapComplete = bootstrapped
		if _, accepted, eulaErr := h.svc.GetEulaStatus(ctx); eulaErr != nil {
			return out, eulaErr
		} else {
			out.EulaVersion = legal.Version
			out.EulaAccepted = accepted
			out.EulaRequired = !accepted
		}
		pool := h.svc.GetPool()
		if pool == nil {
			return out, nil
		}
		licRow, err := billingdb.New(pool).GetLicenseStatus(ctx)
		if err == pgx.ErrNoRows {
			return out, nil
		}
		if err != nil {
			return out, err
		}
		var validUntil time.Time
		hasValidUntil := licRow.ValidUntil.Valid
		if hasValidUntil {
			validUntil = licRow.ValidUntil.Time
		}
		out.License = adminapi.BuildMetaLicense(
			licRow.State,
			licensing.BannerSeverity(licensing.LicenseState(licRow.State)),
			validUntil,
			hasValidUntil,
		)
		return out, nil
	}
}
