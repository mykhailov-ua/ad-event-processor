package controlplane

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/licensing"

	"github.com/google/uuid"
)

var errLicenseWatcherUnavailable = errors.New("license watcher not configured")

func (s *Service) ApplyLicenseToken(ctx context.Context, token string) error {
	if s == nil {
		return errLicenseWatcherUnavailable
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return errValidation("license token is required")
	}

	claims, err := licensing.VerifyJWTResolved(token)
	if err != nil {
		if errors.Is(err, licensing.ErrInvalidSignature) || errors.Is(err, licensing.ErrInvalidTokenFormat) {
			return errValidation("invalid license token")
		}
		return err
	}
	if err := licensing.CheckHostActivation(ctx, s.GetPool(), claims, licensing.HostFingerprint()); err != nil {
		switch {
		case errors.Is(err, licensing.ErrFingerprintMismatch),
			errors.Is(err, licensing.ErrFingerprintRequired),
			errors.Is(err, licensing.ErrActivationLimit):
			return errValidation(err.Error())
		default:
			return err
		}
	}

	path := config.LicensePathFromEnv()
	if err := licensing.InstallToken(path, token, nil); err != nil {
		if errors.Is(err, licensing.ErrInvalidSignature) || errors.Is(err, licensing.ErrInvalidTokenFormat) {
			return errValidation("invalid license token")
		}
		return err
	}
	if err := reloadLicense(ctx); err != nil {
		return err
	}

	s.recordLicenseApplyAudit(ctx, claims)
	return nil
}

func (s *Service) recordLicenseApplyAudit(ctx context.Context, claims *licenseClaims) {
	if s == nil || claims == nil {
		return
	}
	var adminID uuid.UUID
	if u, ok := GetUser(ctx); ok {
		adminID = u.UserID
	}
	depID, err := uuid.Parse(claims.DeploymentID)
	var targetID *uuid.UUID
	if err == nil {
		targetID = &depID
	}
	change := auditLicenseApplyChange{
		DeploymentID: claims.DeploymentID,
		ValidUntil:   claims.ValidUntil.UTC().Format(time.RFC3339),
		CustomerName: claims.CustomerName,
		Plan:         claims.Plan,
		Revoked:      claims.Revoked,
	}
	s.AuditLog(ctx, nil, adminID, "LICENSE_APPLY", "license", targetID, change, nil)
	if s.alerter != nil {
		s.alerter.AlertLicenseApplied(ctx, claims.DeploymentID, claims.ValidUntil, adminID.String(), claims.Revoked)
	}
}

type licenseClaims = licensing.LicenseClaims
