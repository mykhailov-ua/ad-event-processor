package controlplane

import (
	"ad-event-processor/internal/campaign"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/fraudadmin"
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type fraudLabelsAPIAdapter struct {
	svc *Service
}

type fraudLabelsHost struct {
	svc *Service
}

func (h fraudLabelsHost) LabelsPool() *pgxpool.Pool {
	return h.svc.GetPool()
}

func (s *Service) ListMLManualLabelsForCustomer(ctx context.Context, customerID uuid.UUID, limit int) ([]fraudadmin.MLManualLabelDTO, error) {
	return fraudadmin.NewLabels(fraudLabelsHost{svc: s}).ListMLManualLabelsForCustomer(ctx, customerID, limit)
}

func (a fraudLabelsAPIAdapter) ListMLManualLabelsForCustomer(ctx context.Context, customerID uuid.UUID, limit int) ([]fraudadmin.MLManualLabelDTO, error) {
	return fraudadmin.NewLabels(fraudLabelsHost{svc: a.svc}).ListMLManualLabelsForCustomer(ctx, customerID, limit)
}

func (a fraudLabelsAPIAdapter) UpsertMLManualLabelForCustomer(ctx context.Context, customerID uuid.UUID, ipHash string, label int, reason string) error {
	return fraudadmin.NewLabels(fraudLabelsHost{svc: a.svc}).UpsertMLManualLabelForCustomer(ctx, customerID, ipHash, label, reason)
}

func (a fraudLabelsAPIAdapter) BulkUpsertMLManualLabelsForCustomer(ctx context.Context, customerID uuid.UUID, rows []fraudadmin.FraudManualLabelRow) (int, error) {
	return fraudadmin.NewLabels(fraudLabelsHost{svc: a.svc}).BulkUpsertMLManualLabelsForCustomer(ctx, customerID, rows)
}

type fraudDecisionsAPIAdapter struct {
	svc *Service
}

func (a fraudDecisionsAPIAdapter) ExplainFraudDecision(ctx context.Context, customerID uuid.UUID, ipHash string, campaignID *uuid.UUID, hours int) (fraudadmin.FraudDecisionDTO, error) {
	return fraudadmin.ExplainFraudDecision(ctx, fraudDecisionsHost{svc: a.svc}, customerID, ipHash, campaignID, hours)
}

func mapFraudadminErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, fraudadmin.ErrValidation) {
		return errValidation(err.Error())
	}
	if errors.Is(err, fraudadmin.ErrFraudDecisionNotFound) {
		return ErrFraudDecisionNotFound
	}
	return err
}

type fraudIntegrationsAPIAdapter struct {
	svc *Service
}

func (a fraudIntegrationsAPIAdapter) ListFraudIntegrationsForCustomer(ctx context.Context, customerID uuid.UUID) ([]fraudadmin.FraudIntegrationDTO, error) {
	rows, err := fraudadmin.ListIntegrationsForCustomer(ctx, a.svc.GetPool(), customerID)
	if err != nil {
		return nil, mapFraudadminErr(err)
	}
	return rows, nil
}

type fraudOverridesAPIAdapter struct {
	svc *Service
}

func (a fraudOverridesAPIAdapter) ApplyFraudScoringOverrideForCustomer(ctx context.Context, customerID uuid.UUID, req fraudadmin.FraudOverrideRequest) error {
	return mapFraudadminErr(fraudadmin.ApplyFraudScoringOverrideForCustomer(ctx, fraudOverridesHost{svc: a.svc}, customerID, req))
}

type fraudPresetsAPIAdapter struct {
	svc *Service
}

func (a fraudPresetsAPIAdapter) ListFraudPolicyPresets(ctx context.Context) ([]fraudadmin.FraudPolicyPresetDTO, error) {
	return fraudadmin.ListPolicyPresets(ctx, a.svc.GetPool())
}

func (a fraudPresetsAPIAdapter) UpdateFraudPolicyPreset(ctx context.Context, name string, req fraudadmin.PatchFraudPolicyPresetRequest) (fraudadmin.FraudPolicyPresetDTO, error) {
	out, err := fraudadmin.UpdatePolicyPreset(ctx, fraudPresetsHost{svc: a.svc}, name, req)
	if err != nil {
		return fraudadmin.FraudPolicyPresetDTO{}, mapFraudadminErr(err)
	}
	return out, nil
}

type fraudPresetsHost struct {
	svc *Service
}

func (h fraudPresetsHost) PresetsPool() *pgxpool.Pool {
	return h.svc.GetPool()
}

func (h fraudPresetsHost) PresetActorID(ctx context.Context) uuid.UUID {
	if u, ok := GetUser(ctx); ok {
		return u.UserID
	}
	return uuid.Nil
}

func (h fraudPresetsHost) PresetAuditUpdate(ctx context.Context, q db.Querier, adminID uuid.UUID, name string, pass, suspect, ivt, block uint8) {
	h.svc.AuditLog(ctx, q, adminID, "UPDATE_FRAUD_POLICY_PRESET", "system", nil, map[string]any{
		"name":    name,
		"pass":    pass,
		"suspect": suspect,
		"ivt":     ivt,
		"block":   block,
	}, nil)
}

type campaignFraudAPIAdapter struct {
	svc *Service
}

func (a campaignFraudAPIAdapter) GetCampaignFraudConfig(ctx context.Context, campaignID uuid.UUID) (campaign.CampaignFraudConfigDTO, error) {
	return fraudadmin.GetCampaignFraudConfig(ctx, fraudCampaignConfigHost{svc: a.svc}, campaignID)
}

func (a campaignFraudAPIAdapter) UpdateCampaignFraudConfig(ctx context.Context, campaignID uuid.UUID, req campaign.PatchCampaignFraudRequest) (campaign.CampaignFraudConfigDTO, error) {
	out, err := fraudadmin.UpdateCampaignFraudConfig(ctx, fraudCampaignConfigHost{svc: a.svc}, campaignID, req)
	return out, mapFraudadminErr(err)
}

func (a campaignFraudAPIAdapter) PreviewCampaignFraudImpact(ctx context.Context, campaignID uuid.UUID, req campaign.PreviewCampaignFraudRequest) (campaign.CampaignFraudPreviewDTO, error) {
	out, err := fraudadmin.PreviewCampaignFraudImpact(ctx, fraudCampaignConfigHost{svc: a.svc}, campaignID, req)
	return out, mapFraudadminErr(err)
}
