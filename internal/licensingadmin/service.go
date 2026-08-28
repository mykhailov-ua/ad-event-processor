package licensingadmin

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/licensing"
)

type Service struct {
	host Host
}

func NewService(host Host) *Service {
	return &Service{host: host}
}

func (s *Service) ApplyLicenseToken(ctx context.Context, token string) error {
	if s == nil || s.host == nil {
		return ErrLicenseWatcherUnavailable
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return s.host.ErrValidation("license token is required")
	}

	claims, err := licensing.VerifyJWTResolved(token)
	if err != nil {
		if errors.Is(err, licensing.ErrInvalidSignature) || errors.Is(err, licensing.ErrInvalidTokenFormat) {
			return s.host.ErrValidation("invalid license token")
		}
		return err
	}
	if err := licensing.CheckHostActivation(ctx, s.host.Pool(), claims, licensing.HostFingerprint()); err != nil {
		switch {
		case errors.Is(err, licensing.ErrFingerprintMismatch),
			errors.Is(err, licensing.ErrFingerprintRequired),
			errors.Is(err, licensing.ErrActivationLimit):
			return s.host.ErrValidation(err.Error())
		default:
			return err
		}
	}

	path := config.LicensePathFromEnv()
	if err := licensing.InstallToken(path, token, nil); err != nil {
		if errors.Is(err, licensing.ErrInvalidSignature) || errors.Is(err, licensing.ErrInvalidTokenFormat) {
			return s.host.ErrValidation("invalid license token")
		}
		return err
	}
	if err := s.host.ReloadLicense(ctx); err != nil {
		return err
	}

	s.host.AuditLicenseApply(ctx, claims)
	return nil
}

func (s *Service) EnforceDeploymentCampaignCap(ctx context.Context) error {
	if s == nil || s.host == nil || s.host.Pool() == nil {
		return nil
	}
	limits, state, ok := s.host.DeploymentLimits()
	if !ok {
		return nil
	}
	if state == licensing.StateExpired || state == licensing.StateRevoked {
		return s.host.ErrValidation("license not active")
	}
	maxActive := limits.MaxActiveCampaigns
	if maxActive == 0 {
		return nil
	}
	var active int64
	err := s.host.Pool().QueryRow(ctx, `
		SELECT COUNT(*) FROM campaigns WHERE status = 'ACTIVE'`).Scan(&active)
	if err != nil {
		return fmt.Errorf("count deployment active campaigns: %w", err)
	}
	if uint64(active) >= maxActive {
		return ErrDeploymentCampaignLimit
	}
	return nil
}
