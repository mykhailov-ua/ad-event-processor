package controlplane

import (
	"context"
	"errors"
	"os"

	"ad-event-processor/pkg/naming"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/telemetry"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *Service) telemetryMetadata(ctx context.Context) (telemetry.Metadata, error) {
	meta := telemetry.Metadata{
		BinaryVersion: os.Getenv(naming.LegacyVendorEnvKey("BINARY_VERSION")),
		DCRegion:      os.Getenv(naming.LegacyVendorEnvKey("DC_REGION")),
	}
	if meta.BinaryVersion == "" {
		meta.BinaryVersion = "dev"
	}
	if s == nil || s.GetPool() == nil {
		return meta, nil
	}
	var deploymentID uuid.UUID
	err := s.GetPool().QueryRow(ctx, `
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

func (s *Service) StartProductTelemetryPulse() {
	if s == nil || s.cfg == nil || !s.cfg.TelemetryOptIn {
		return
	}
	worker := telemetry.NewWorker(telemetry.Config{
		OptIn:            s.cfg.TelemetryOptIn,
		URL:              string(s.cfg.TelemetryURL),
		LicenseServerURL: config.LicenseEnv("SERVER"),
		Interval:         s.cfg.TelemetryInterval(),
		HTTPTimeout:      s.cfg.TelemetryHTTPTimeout(),
		Metadata:         s.telemetryMetadata,
	})
	if !worker.Enabled() {
		return
	}
	s.startWorker(func() {
		worker.Start(s.ctx)
	})
}
