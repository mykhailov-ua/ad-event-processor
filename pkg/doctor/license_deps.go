package doctor

import (
	"sync"
	"time"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/licensing"
)

type BundleLicenseDTO struct {
	FingerprintMatch bool   `json:"fingerprint_match"`
	HWIDv2           string `json:"hwid_v2,omitempty"`
	HWIDMatch        *bool  `json:"hwid_match,omitempty"`
	DeploymentID     string `json:"deployment_id,omitempty"`
	DaysToExpiry     int    `json:"days_to_expiry"`
	State            string `json:"state"`
}

func WithCLILicenseDeps(deps ProbeDeps) ProbeDeps {
	if !config.LicenseProbeEnabled() {
		return deps
	}
	if deps.LicenseState != nil && deps.LicenseDiagnostics != nil {
		return deps
	}
	stateFn, diagFn := cliLicenseSnapshotFns(deps.Config)
	if deps.LicenseState == nil {
		deps.LicenseState = stateFn
	}
	if deps.LicenseDiagnostics == nil {
		deps.LicenseDiagnostics = diagFn
	}
	return deps
}

func bundleLicenseInfo(deps ProbeDeps) BundleLicenseDTO {
	if deps.LicenseDiagnostics == nil {
		return BundleLicenseDTO{State: string(licensing.StateExpired)}
	}
	diag, ok := deps.LicenseDiagnostics()
	if !ok {
		return BundleLicenseDTO{State: string(licensing.StateExpired)}
	}
	return BundleLicenseDTO{
		FingerprintMatch: diag.FingerprintMatch,
		HWIDv2:           diag.HostHWID,
		HWIDMatch:        hwidMatchPtr(diag),
		DeploymentID:     diag.DeploymentID,
		DaysToExpiry:     diag.DaysToExpiry,
		State:            string(diag.State),
	}
}

func hwidMatchPtr(diag licensing.LicenseDiagnostics) *bool {
	if !licensing.BindModeHard(diag.BindMode) {
		return nil
	}
	if diag.BindHWIDHash == "" && diag.BindFingerprint == "" {
		return nil
	}
	match := diag.HWIDMatch
	return &match
}

func cliLicenseSnapshotFns(cfg *config.Config) (func() (licensing.LicenseState, bool), func() (licensing.LicenseDiagnostics, bool)) {
	var once sync.Once
	var diag licensing.LicenseDiagnostics
	var state licensing.LicenseState
	var ready bool

	load := func() {
		once.Do(func() {
			diag, state, ready = loadCLILicense(cfg)
		})
	}
	stateFn := func() (licensing.LicenseState, bool) {
		load()
		return state, ready
	}
	diagFn := func() (licensing.LicenseDiagnostics, bool) {
		load()
		return diag, ready
	}
	return stateFn, diagFn
}

func loadCLILicense(cfg *config.Config) (licensing.LicenseDiagnostics, licensing.LicenseState, bool) {
	_ = cfg
	path := config.LicensePathFromEnv()
	verified, err := licensing.VerifyLicenseFile(path, nil, licensing.HostFingerprint(), time.Now())
	if err != nil {
		return licensing.LicenseDiagnostics{
			State: licensing.StateExpired,
		}, licensing.StateExpired, true
	}
	diag := licensing.BuildLicenseDiagnostics(verified.Claims, verified.State, time.Now())
	return diag, verified.State, true
}

func licenseProbe(deps ProbeDeps) Probe {
	return LicenseProbe{
		StateFn:       deps.LicenseState,
		DiagnosticsFn: deps.LicenseDiagnostics,
	}
}
