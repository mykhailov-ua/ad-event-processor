package trialregistry

import "os"

const EnvTrialSelfServeURL = "VENDOR_TRIAL_SELF_SERVE_URL"

func SelfServeURL() string {
	return os.Getenv(EnvTrialSelfServeURL)
}
