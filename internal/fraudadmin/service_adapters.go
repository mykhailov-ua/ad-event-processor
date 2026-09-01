package fraudadmin

import (
	"context"

	"ad-event-processor/internal/campaign"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MapErrorFunc func(error) error

type LabelsAPI struct {
	Host LabelsHost
}

func (a LabelsAPI) ListMLManualLabelsForCustomer(ctx context.Context, customerID uuid.UUID, limit, offset int) ([]MLManualLabelDTO, int64, error) {
	return NewLabels(a.Host).ListMLManualLabelsForCustomer(ctx, customerID, limit, offset)
}

func (a LabelsAPI) UpsertMLManualLabelForCustomer(ctx context.Context, customerID uuid.UUID, ipHash string, label int, reason string) error {
	return NewLabels(a.Host).UpsertMLManualLabelForCustomer(ctx, customerID, ipHash, label, reason)
}

func (a LabelsAPI) BulkUpsertMLManualLabelsForCustomer(ctx context.Context, customerID uuid.UUID, rows []FraudManualLabelRow) (int, error) {
	return NewLabels(a.Host).BulkUpsertMLManualLabelsForCustomer(ctx, customerID, rows)
}

type DecisionsAPI struct {
	Host DecisionsHost
}

func (a DecisionsAPI) ExplainFraudDecision(ctx context.Context, customerID uuid.UUID, ipHash string, campaignID *uuid.UUID, hours int) (FraudDecisionDTO, error) {
	return ExplainFraudDecision(ctx, a.Host, customerID, ipHash, campaignID, hours)
}

type IntegrationsAPI struct {
	Pool   *pgxpool.Pool
	MapErr MapErrorFunc
}

func (a IntegrationsAPI) ListFraudIntegrationsForCustomer(ctx context.Context, customerID uuid.UUID) ([]FraudIntegrationDTO, error) {
	rows, err := ListIntegrationsForCustomer(ctx, a.Pool, customerID)
	if a.MapErr != nil {
		err = a.MapErr(err)
	}
	return rows, err
}

type OverridesAPI struct {
	Host   OverridesHost
	MapErr MapErrorFunc
}

func (a OverridesAPI) ApplyFraudScoringOverrideForCustomer(ctx context.Context, customerID uuid.UUID, req FraudOverrideRequest) error {
	err := ApplyFraudScoringOverrideForCustomer(ctx, a.Host, customerID, req)
	if a.MapErr != nil {
		return a.MapErr(err)
	}
	return err
}

type PresetsAPI struct {
	Host   PresetsHost
	Pool   *pgxpool.Pool
	MapErr MapErrorFunc
}

func (a PresetsAPI) ListFraudPolicyPresets(ctx context.Context) ([]FraudPolicyPresetDTO, error) {
	return ListPolicyPresets(ctx, a.Pool)
}

func (a PresetsAPI) UpdateFraudPolicyPreset(ctx context.Context, name string, req PatchFraudPolicyPresetRequest) (FraudPolicyPresetDTO, error) {
	out, err := UpdatePolicyPreset(ctx, a.Host, name, req)
	if a.MapErr != nil {
		err = a.MapErr(err)
	}
	if err != nil {
		return FraudPolicyPresetDTO{}, err
	}
	return out, nil
}

type CampaignFraudAPI struct {
	Host   CampaignConfigHost
	MapErr MapErrorFunc
}

func (a CampaignFraudAPI) GetCampaignFraudConfig(ctx context.Context, campaignID uuid.UUID) (campaign.CampaignFraudConfigDTO, error) {
	return GetCampaignFraudConfig(ctx, a.Host, campaignID)
}

func (a CampaignFraudAPI) UpdateCampaignFraudConfig(ctx context.Context, campaignID uuid.UUID, req campaign.PatchCampaignFraudRequest) (campaign.CampaignFraudConfigDTO, error) {
	out, err := UpdateCampaignFraudConfig(ctx, a.Host, campaignID, req)
	if a.MapErr != nil {
		err = a.MapErr(err)
	}
	return out, err
}

func (a CampaignFraudAPI) PreviewCampaignFraudImpact(ctx context.Context, campaignID uuid.UUID, req campaign.PreviewCampaignFraudRequest) (campaign.CampaignFraudPreviewDTO, error) {
	out, err := PreviewCampaignFraudImpact(ctx, a.Host, campaignID, req)
	if a.MapErr != nil {
		err = a.MapErr(err)
	}
	return out, err
}

var (
	_ LabelsService                 = LabelsAPI{}
	_ DecisionsService              = DecisionsAPI{}
	_ IntegrationsService           = IntegrationsAPI{}
	_ OverridesService              = OverridesAPI{}
	_ PresetsService                = PresetsAPI{}
	_ campaign.CampaignFraudService = CampaignFraudAPI{}
)
