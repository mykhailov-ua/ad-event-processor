package controlplane

import "ad-event-processor/internal/fraudadmin"

type (
	FraudHTTPHandlers         = fraudadmin.HTTPHandlers
	FraudDecisionDTO          = fraudadmin.FraudDecisionDTO
	FraudIntegrationDTO       = fraudadmin.FraudIntegrationDTO
	FraudOverrideRequest      = fraudadmin.FraudOverrideRequest
	FraudPolicyPresetDTO      = fraudadmin.FraudPolicyPresetDTO
	PatchFraudPolicyPresetRequest = fraudadmin.PatchFraudPolicyPresetRequest
	FraudTierThresholdsDTO    = fraudadmin.FraudTierThresholdsDTO
	MLManualLabelDTO          = fraudadmin.MLManualLabelDTO
	FraudManualLabelRow       = fraudadmin.FraudManualLabelRow
	FraudManualLabelRequest   = fraudadmin.FraudManualLabelRequest
	MLManualLabelRequest      = fraudadmin.FraudManualLabelRequest
	FraudLabelsService        = fraudadmin.LabelsService
	FraudDecisionsService     = fraudadmin.DecisionsService
	FraudIntegrationsService  = fraudadmin.IntegrationsService
	FraudOverridesService     = fraudadmin.OverridesService
	FraudPresetsService       = fraudadmin.PresetsService
)

func FraudDecisionDisclaimer() string {
	return fraudadmin.DecisionDisclaimer
}
