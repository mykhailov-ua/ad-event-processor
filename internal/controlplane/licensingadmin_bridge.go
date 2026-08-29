package controlplane

import (
	"context"
	"time"

	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/licensing"
	"ad-event-processor/internal/licensingadmin"
	"ad-event-processor/internal/platformadmin"
	"ad-event-processor/pkg/legal"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type licensingHost struct {
	svc *Service
}

func (s *Service) LicensingService() *licensingadmin.Service {
	if s == nil {
		return nil
	}
	return licensingadmin.NewService(licensingHost{svc: s})
}

func (h licensingHost) Pool() *pgxpool.Pool { return h.svc.GetPool() }

func (h licensingHost) ReloadLicense(ctx context.Context) error { return reloadLicense(ctx) }

func (h licensingHost) ActiveActivationLicenseKey() string { return activeActivationLicenseKey() }

func (h licensingHost) WorkerOpContext(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return workerContext(parent, timeout)
}

func (h licensingHost) ErrValidation(msg string) error { return errValidation(msg) }

func (h licensingHost) ActorUserID(ctx context.Context) uuid.UUID {
	if u, ok := GetUser(ctx); ok {
		return u.UserID
	}
	return uuid.Nil
}

func (h licensingHost) AuditLicenseApply(ctx context.Context, claims *licensing.LicenseClaims) {
	if h.svc == nil || claims == nil {
		return
	}
	adminID := h.ActorUserID(ctx)
	depID, err := uuid.Parse(claims.DeploymentID)
	var targetID *uuid.UUID
	if err == nil {
		targetID = &depID
	}
	change := platformadmin.AuditLicenseApplyChange{
		DeploymentID: claims.DeploymentID,
		ValidUntil:   claims.ValidUntil.UTC().Format(time.RFC3339),
		CustomerName: claims.CustomerName,
		Plan:         claims.Plan,
		Revoked:      claims.Revoked,
	}
	h.svc.AuditLog(ctx, nil, adminID, "LICENSE_APPLY", "license", targetID, change, nil)
	if h.svc.alerter != nil {
		h.svc.alerter.AlertLicenseApplied(ctx, claims.DeploymentID, claims.ValidUntil, adminID.String(), claims.Revoked)
	}
}

func (h licensingHost) DeploymentLimits() (licensing.Limits, licensing.LicenseState, bool) {
	return licenseDeploymentLimits()
}

func (h licensingHost) FeatureAllowed(featureKey string) (bool, string) {
	return licenseFeatureAllowed(featureKey)
}

func (h licensingHost) EulaPool() *pgxpool.Pool { return h.svc.GetPool() }

func (h licensingHost) EulaActorID(ctx context.Context) uuid.UUID { return h.ActorUserID(ctx) }

func (h licensingHost) EulaAuditAccept(ctx context.Context, q db.Querier, adminID uuid.UUID, version, acceptedBy string) {
	h.svc.AuditLog(ctx, q, adminID, "EULA_ACCEPT", "system", nil, map[string]string{
		"version": version,
		"by":      acceptedBy,
	}, nil)
}

func (s *Service) ApplyLicenseToken(ctx context.Context, token string) error {
	return s.LicensingService().ApplyLicenseToken(ctx, token)
}

func (s *Service) enforceDeploymentLicenseCampaignCap(ctx context.Context) error {
	return s.LicensingService().EnforceDeploymentCampaignCap(ctx)
}

func (s *Service) StartLicenseRevokeQueueWorker(interval time.Duration) {
	if s == nil || s.pool == nil {
		return
	}
	s.startWorker(func() {
		licensingadmin.NewRevokeQueueWorker(
			s.pool,
			interval,
			reloadLicense,
			activeActivationLicenseKey,
		).Start(s.ctx)
	})
}

func (s *Service) GetEulaStatus(ctx context.Context) (legal.Acceptance, bool, error) {
	return licensingadmin.GetEulaStatus(ctx, licensingHost{svc: s})
}

func (s *Service) AcceptEula(ctx context.Context, version, acceptedBy string) error {
	return licensingadmin.AcceptEula(ctx, licensingHost{svc: s}, version, acceptedBy)
}

func (s *Service) saveEulaAcceptanceTx(ctx context.Context, q db.Querier, version, acceptedBy string) error {
	return licensingadmin.SaveEulaAcceptanceTx(ctx, licensingHost{svc: s}, q, version, acceptedBy)
}

var _ licensingadmin.EulaHost = licensingHost{}
