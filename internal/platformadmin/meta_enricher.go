package platformadmin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	billingdb "ad-event-processor/internal/ledger/db"
	"ad-event-processor/internal/licensing"
	"ad-event-processor/pkg/legal"
	"ad-event-processor/pkg/platformconfig"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MetaEnrichHost interface {
	SupportFeedbackMeta(ctx context.Context) (SupportFeedbackMeta, error)
	GetPlatformConfig(ctx context.Context) (platformconfig.Config, bool, error)
	GetEulaStatus(ctx context.Context) (legal.Acceptance, bool, error)
	Pool() *pgxpool.Pool
}

func NewMetaEnricher(host MetaEnrichHost) MetaEnricher {
	return func(ctx context.Context) (MetaEnrichOut, error) {
		out := MetaEnrichOut{}
		if host == nil {
			return out, nil
		}
		if fbMeta, err := host.SupportFeedbackMeta(ctx); err != nil {
			return out, err
		} else {
			out.DeploymentID = fbMeta.DeploymentID
		}
		_, bootstrapped, err := host.GetPlatformConfig(ctx)
		if err != nil {
			return out, err
		}
		out.BootstrapComplete = bootstrapped
		if _, accepted, eulaErr := host.GetEulaStatus(ctx); eulaErr != nil {
			return out, eulaErr
		} else {
			out.EulaVersion = legal.Version
			out.EulaAccepted = accepted
			out.EulaRequired = !accepted
		}
		pool := host.Pool()
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
