package verify

import entitlements "ad-event-processor/internal/licensing/entitlements"

type (
	Entitlements  = entitlements.Entitlements
	FeatureSet    = entitlements.FeatureSet
	FeatureSetDTO = entitlements.FeatureSetDTO
	LicenseClaims = entitlements.LicenseClaims
	LicenseState  = entitlements.LicenseState
	Limits        = entitlements.Limits
	LimitsDTO     = entitlements.LimitsDTO
)

var (
	DetermineState          = entitlements.DetermineState
	Effective               = entitlements.Effective
	FeatureSeedValid        = entitlements.FeatureSeedValid
	IngestAllowed           = entitlements.IngestAllowed
	LicenseEpochInvalid     = entitlements.LicenseEpochInvalid
	PublishFeatureSeed      = entitlements.PublishFeatureSeed
	ResetFeatureSeedForTest = entitlements.ResetFeatureSeedForTest
	StateActive             = entitlements.StateActive
	StateExpired            = entitlements.StateExpired
	StateGrace              = entitlements.StateGrace
	StateOfflineGrace       = entitlements.StateOfflineGrace
	StateOfflineWarn        = entitlements.StateOfflineWarn
	StateRevoked            = entitlements.StateRevoked
)
