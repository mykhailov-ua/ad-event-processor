package licensingadmin

import (
	"context"
	"time"

	"ad-event-processor/internal/licensing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Host interface {
	Pool() *pgxpool.Pool
	ReloadLicense(ctx context.Context) error
	ActiveActivationLicenseKey() string
	WorkerOpContext(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc)
	ErrValidation(msg string) error
	ActorUserID(ctx context.Context) uuid.UUID
	AuditLicenseApply(ctx context.Context, claims *licensing.LicenseClaims)
	DeploymentLimits() (licensing.Limits, licensing.LicenseState, bool)
	FeatureAllowed(featureKey string) (allowed bool, planCode string)
}
