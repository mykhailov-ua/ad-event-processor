package licensingadmin

import (
	"net/http"

	"ad-event-processor/pkg/httpresponse"
)

type FeatureChecker func(featureKey string) (allowed bool, planCode string)

type LicenseFeatureRequiredBody struct {
	Error       string `json:"error"`
	FeatureKey  string `json:"feature_key"`
	PlanCode    string `json:"plan_code,omitempty"`
	FeatureGate string `json:"feature_required"`
}

func FeatureAllowed(check FeatureChecker, featureKey string) (allowed bool, planCode string) {
	if check == nil {
		return true, ""
	}
	return check(featureKey)
}

func WriteLicenseFeatureRequired(w http.ResponseWriter, featureKey, planCode string) {
	httpresponse.JSON(w, http.StatusForbidden, LicenseFeatureRequiredBody{
		Error:       "feature_required",
		FeatureKey:  featureKey,
		PlanCode:    planCode,
		FeatureGate: featureKey,
	})
}

func RequireLicenseFeature(w http.ResponseWriter, check FeatureChecker, featureKey string) bool {
	allowed, planCode := FeatureAllowed(check, featureKey)
	if allowed {
		return true
	}
	WriteLicenseFeatureRequired(w, featureKey, planCode)
	return false
}