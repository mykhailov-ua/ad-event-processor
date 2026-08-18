package controlplane

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	billingdb "github.com/bidshard/ad-event-processor/internal/ledger/db"
	"github.com/bidshard/ad-event-processor/internal/licensing"
	"github.com/bidshard/ad-event-processor/pkg/legal"

	"github.com/jackc/pgx/v5"
)

func (h *Handler) metaEnricher() MetaEnricher {
	return func(ctx context.Context) (MetaEnrichOut, error) {
		out := MetaEnrichOut{}
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
		if errors.Is(err, pgx.ErrNoRows) {
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
		var ent licensing.Entitlements
		if len(licRow.EntitlementsJson) > 0 {
			if err := json.Unmarshal(licRow.EntitlementsJson, &ent); err != nil {
				return out, fmt.Errorf("decode license entitlements: %w", err)
			}
		}
		var activeCampaigns int64
		if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM campaigns WHERE status = 'ACTIVE'`).Scan(&activeCampaigns); err != nil {
			return out, err
		}
		policy := licensing.LoadHeartbeatPolicyFromEnv()
		warnings := licensing.TierUsageWarnings(
			ent.Limits,
			int(activeCampaigns),
			licensing.LicenseState(licRow.State),
			validUntil,
			time.Now(),
			policy.RenewBeforeDays,
		)
		out.License = BuildMetaLicense(MetaLicenseBuildInput{
			State:          licRow.State,
			BannerSeverity: licensing.BannerSeverity(licensing.LicenseState(licRow.State)),
			PlanCode:       licRow.PlanCode,
			ValidUntil:     validUntil,
			HasValidUntil:  hasValidUntil,
			TierWarnings:   warnings,
		})
		return out, nil
	}
}
