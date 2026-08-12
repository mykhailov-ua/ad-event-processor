package doctor

import (
	"context"
	"fmt"
	"time"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/licensing"
)

type LicenseProbe struct {
	StateFn       func() (licensing.LicenseState, bool)
	DiagnosticsFn func() (licensing.LicenseDiagnostics, bool)
}

func (LicenseProbe) Name() string { return "license" }

func (p LicenseProbe) Run(ctx context.Context) Result {
	start := time.Now()
	if !config.LicenseProbeEnabled() {
		return Result{Name: "license", Status: StatusSkip, Detail: "license not required and no license file", Latency: time.Since(start).Milliseconds()}
	}
	if p.StateFn == nil {
		return Result{Name: "license", Status: StatusFail, Detail: "license watcher not configured", Latency: time.Since(start).Milliseconds()}
	}
	state, ok := p.StateFn()
	if !ok {
		return Result{Name: "license", Status: StatusFail, Detail: "license watcher not configured", Latency: time.Since(start).Milliseconds()}
	}
	latency := time.Since(start).Milliseconds()

	var diag licensing.LicenseDiagnostics
	if p.DiagnosticsFn != nil {
		diag, _ = p.DiagnosticsFn()
	}

	if !licensing.IngestAllowed(state) {
		return Result{
			Name:    "license",
			Status:  StatusFail,
			Detail:  licenseDetail(state, diag),
			Latency: latency,
		}
	}
	status := StatusPass
	detail := licenseDetail(state, diag)
	if diag.BindMode != "" && licensing.BindModeHard(diag.BindMode) && !diag.FingerprintMatch {
		status = StatusFail
		detail += "; fingerprint mismatch"
	} else if diag.DaysToExpiry <= 7 && diag.DaysToExpiry >= 0 {
		status = StatusWarn
	}
	return Result{
		Name:    "license",
		Status:  status,
		Detail:  detail,
		Latency: latency,
	}
}

func licenseDetail(state licensing.LicenseState, diag licensing.LicenseDiagnostics) string {
	if diag.DeploymentID != "" {
		if diag.DaysToExpiry > 0 || diag.ValidUntil.IsZero() {
			return fmt.Sprintf("state=%s deployment_id=%s days_to_expiry=%d fingerprint_match=%v",
				state, diag.DeploymentID, diag.DaysToExpiry, diag.FingerprintMatch)
		}
		return fmt.Sprintf("state=%s deployment_id=%s valid_until=%s fingerprint_match=%v",
			state, diag.DeploymentID, diag.ValidUntil.UTC().Format(time.RFC3339), diag.FingerprintMatch)
	}
	return fmt.Sprintf("state=%s", state)
}
