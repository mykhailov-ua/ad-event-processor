package management

import (
	"context"
	"os"

	"espx/internal/telemetry"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *Service) telemetryMetadata(ctx context.Context) (telemetry.Metadata, error) {
	meta := telemetry.Metadata{
		BinaryVersion: os.Getenv("ESPX_BINARY_VERSION"),
		DCRegion:      os.Getenv("ESPX_DC_REGION"),
	}
	if meta.BinaryVersion == "" {
		meta.BinaryVersion = "dev"
	}
	if s == nil || s.GetPool() == nil {
		return meta, nil
	}
	var deploymentID uuid.UUID
	var planCode string
	err := s.GetPool().QueryRow(ctx, `
		SELECT deployment_id, plan_code
		FROM billing.license_status
		LIMIT 1`).Scan(&deploymentID, &planCode)
	if err != nil {
		if err == pgx.ErrNoRows {
			return meta, nil
		}
		return meta, err
	}
	if deploymentID != uuid.Nil {
		meta.DeploymentID = deploymentID.String()
	}
	meta.SKU = planCode
	return meta, nil
}

func (s *Service) StartProductTelemetryPulse() {
	if s == nil || s.cfg == nil || !s.cfg.TelemetryOptIn {
		return
	}
	worker := telemetry.NewWorker(telemetry.Config{
		OptIn:            s.cfg.TelemetryOptIn,
		URL:              string(s.cfg.TelemetryURL),
		LicenseServerURL: os.Getenv("ESPX_LICENSE_SERVER"),
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
