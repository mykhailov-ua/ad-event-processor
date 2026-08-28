package platformadmin

import (
	"context"
	"errors"
	"os"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/telemetry"
	"ad-event-processor/pkg/naming"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ProductTelemetryHost interface {
	TelemetryOptIn() bool
	TelemetryURL() string
	TelemetryInterval() time.Duration
	TelemetryHTTPTimeout() time.Duration
	Pool() *pgxpool.Pool
	StartWorker(fn func())
	WorkerContext() context.Context
}

func TelemetryMetadata(ctx context.Context, pool *pgxpool.Pool) (telemetry.Metadata, error) {
	meta := telemetry.Metadata{
		BinaryVersion: os.Getenv(naming.LegacyVendorEnvKey("BINARY_VERSION")),
		DCRegion:      os.Getenv(naming.LegacyVendorEnvKey("DC_REGION")),
	}
	if meta.BinaryVersion == "" {
		meta.BinaryVersion = "dev"
	}
	if pool == nil {
		return meta, nil
	}
	var deploymentID uuid.UUID
	err := pool.QueryRow(ctx, `
		SELECT deployment_id
		FROM billing.license_status
		LIMIT 1`).Scan(&deploymentID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return meta, nil
		}
		return meta, err
	}
	if deploymentID != uuid.Nil {
		meta.DeploymentID = deploymentID.String()
	}
	return meta, nil
}

func StartProductTelemetryPulse(host ProductTelemetryHost) {
	if host == nil || !host.TelemetryOptIn() {
		return
	}
	worker := telemetry.NewWorker(telemetry.Config{
		OptIn:            host.TelemetryOptIn(),
		URL:              host.TelemetryURL(),
		LicenseServerURL: config.LicenseEnv("SERVER"),
		Interval:         host.TelemetryInterval(),
		HTTPTimeout:      host.TelemetryHTTPTimeout(),
		Metadata: func(ctx context.Context) (telemetry.Metadata, error) {
			return TelemetryMetadata(ctx, host.Pool())
		},
	})
	if !worker.Enabled() {
		return
	}
	host.StartWorker(func() {
		worker.Start(host.WorkerContext())
	})
}
