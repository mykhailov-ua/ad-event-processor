package fraudadmin

import (
	"context"

	"github.com/google/uuid"
)

type MLManualLabelDTO struct {
	IPHash    string `json:"ip_hash"`
	Label     int    `json:"label"`
	Reason    string `json:"reason"`
	Source    string `json:"source"`
	CreatedAt string `json:"created_at"`
}

type FraudManualLabelRow struct {
	IPHash string `json:"ip_hash"`
	Label  int    `json:"label"`
	Reason string `json:"reason"`
}

type FraudManualLabelRequest struct {
	IPHash string `json:"ip_hash"`
	Label  int    `json:"label"`
	Reason string `json:"reason"`
}

type FraudManualLabelBulkRequest struct {
	Rows []FraudManualLabelRow `json:"rows"`
}

type FraudManualLabelBulkResponse struct {
	Upserted int `json:"upserted"`
}

type MLManualLabelRequest = FraudManualLabelRequest

type FraudTierThresholdsDTO struct {
	Scope      string `json:"scope,omitempty"`
	PassMax    int    `json:"pass_max"`
	SuspectMax int    `json:"suspect_max"`
	IVTMax     int    `json:"ivt_max"`
	BlockAbove int    `json:"block_above"`
}

type FraudDecisionDTO struct {
	IPHash              string                 `json:"ip_hash"`
	CampaignID          string                 `json:"campaign_id"`
	WindowStart         string                 `json:"window_start"`
	EvaluatedAt         string                 `json:"evaluated_at"`
	Disclaimer          string                 `json:"disclaimer"`
	Tier                string                 `json:"tier"`
	Score               int                    `json:"score"`
	MLProbability       float64                `json:"ml_probability"`
	AdjustedProbability float64                `json:"adjusted_probability"`
	ResidentialProxy    bool                   `json:"residential_proxy"`
	StructuralFraud     bool                   `json:"structural_fraud"`
	FPGuardApplied      bool                   `json:"fp_guard_applied"`
	ModelScore          *float64               `json:"model_score,omitempty"`
	ModelName           string                 `json:"model_name,omitempty"`
	ScoreMissing        bool                   `json:"score_missing"`
	Features            map[string]float64     `json:"features"`
	CampaignThresholds  FraudTierThresholdsDTO `json:"campaign_thresholds"`
}

type FraudIntegrationDTO struct {
	CampaignID    string `json:"campaign_id"`
	Name          string `json:"name"`
	Provider      string `json:"provider,omitempty"`
	Configured    bool   `json:"configured"`
	HealthStatus  string `json:"health_status"`
	LastSuccessAt string `json:"last_success_at,omitempty"`
	DLQCount      int64  `json:"dlq_count"`
	LastError     string `json:"last_error,omitempty"`
}

type FraudOverrideRequest struct {
	CampaignID *string `json:"campaign_id,omitempty"`
	IP         *string `json:"ip,omitempty"`
	IPHash     *string `json:"ip_hash,omitempty"`
}

type FraudScoringOverrideRequest struct {
	CampaignID *string `json:"campaign_id,omitempty"`
	IP         *string `json:"ip,omitempty"`
}

type FraudPolicyPresetDTO struct {
	Name      string `json:"name"`
	Pass      uint8  `json:"pass"`
	Suspect   uint8  `json:"suspect"`
	IVT       uint8  `json:"ivt"`
	Block     uint8  `json:"block"`
	UpdatedAt string `json:"updated_at"`
}

type PatchFraudPolicyPresetRequest struct {
	Pass    *uint8 `json:"pass,omitempty"`
	Suspect *uint8 `json:"suspect,omitempty"`
	IVT     *uint8 `json:"ivt,omitempty"`
	Block   *uint8 `json:"block,omitempty"`
}

type LabelsService interface {
	ListMLManualLabelsForCustomer(ctx context.Context, customerID uuid.UUID, limit int) ([]MLManualLabelDTO, error)
	UpsertMLManualLabelForCustomer(ctx context.Context, customerID uuid.UUID, ipHash string, label int, reason string) error
	BulkUpsertMLManualLabelsForCustomer(ctx context.Context, customerID uuid.UUID, rows []FraudManualLabelRow) (int, error)
}

type DecisionsService interface {
	ExplainFraudDecision(ctx context.Context, customerID uuid.UUID, ipHash string, campaignID *uuid.UUID, hours int) (FraudDecisionDTO, error)
}

type IntegrationsService interface {
	ListFraudIntegrationsForCustomer(ctx context.Context, customerID uuid.UUID) ([]FraudIntegrationDTO, error)
}

type OverridesService interface {
	ApplyFraudScoringOverrideForCustomer(ctx context.Context, customerID uuid.UUID, req FraudOverrideRequest) error
}

type PresetsService interface {
	ListFraudPolicyPresets(ctx context.Context) ([]FraudPolicyPresetDTO, error)
	UpdateFraudPolicyPreset(ctx context.Context, name string, req PatchFraudPolicyPresetRequest) (FraudPolicyPresetDTO, error)
}
