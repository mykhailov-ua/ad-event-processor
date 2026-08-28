package controlplane

import (
	"net/http"

	"ad-event-processor/internal/licensing"
	"ad-event-processor/pkg/httpresponse"
)

type LicenseFeatureRequiredBody struct {
	Error       string `json:"error"`
	FeatureKey  string `json:"feature_key"`
	PlanCode    string `json:"plan_code,omitempty"`
	FeatureGate string `json:"feature_required"`
}

func licenseFeatureAllowed(featureKey string) (allowed bool, planCode string) {
	w := activeLicenseWatcher.Load()
	if w == nil {
		return true, ""
	}
	state, claims := w.GetState()
	if claims == nil {
		return state == licensing.StateActive || state == licensing.StateGrace || state == licensing.StateOfflineWarn, ""
	}
	planCode = claims.Plan
	ent := licensing.Entitlements{Features: claims.Features}
	switch featureKey {
	case "openrtb":
		return licensing.OpenRTBAllowed(state, ent), planCode
	case "fraud_dispute_evidence":
		return licensing.FraudDisputeEvidenceAllowed(state, ent), planCode
	default:
		return true, planCode
	}
}

func writeLicenseFeatureRequired(w http.ResponseWriter, featureKey, planCode string) {
	httpresponse.JSON(w, http.StatusForbidden, LicenseFeatureRequiredBody{
		Error:       "feature_required",
		FeatureKey:  featureKey,
		PlanCode:    planCode,
		FeatureGate: featureKey,
	})
}

func requireLicenseFeature(w http.ResponseWriter, featureKey string) bool {
	allowed, planCode := licenseFeatureAllowed(featureKey)
	if allowed {
		return true
	}
	writeLicenseFeatureRequired(w, featureKey, planCode)
	return false
}
