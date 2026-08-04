package doctor

import (
	"context"
	"strings"

	"espx/pkg/platformconfig"
)

type DoctorCheckDTO struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	Message   string `json:"message"`
	Hint      string `json:"hint,omitempty"`
	LatencyMs int64  `json:"latency_ms"`
}

func RunPlatform(ctx context.Context, deps ProbeDeps, cfg platformconfig.Config, opts Options) Report {
	probes := DefaultProbes(deps)
	probes = append(probes, RtbConfigProbe{Deps: deps})
	if deps.LicenseState != nil {
		probe := LicenseProbe{StateFn: deps.LicenseState}
		if deps.LicenseDiagnostics != nil {
			probe.DiagnosticsFn = deps.LicenseDiagnostics
		}
		probes = append(probes, probe)
	}
	if strings.TrimSpace(cfg.TrackingDomain) != "" {
		probes = append(probes, DNSProbe{Hostname: cfg.TrackingDomain})
	}
	opts.Probes = probes
	return Run(ctx, opts)
}

func ReportToDTO(rep Report) []DoctorCheckDTO {
	out := make([]DoctorCheckDTO, 0, len(rep.Results))
	for _, r := range rep.Results {
		out = append(out, DoctorCheckDTO{
			ID:        r.Name,
			Status:    r.Status.String(),
			Message:   r.Detail,
			Hint:      CheckHint(r.Name),
			LatencyMs: r.Latency,
		})
	}
	return out
}

func OverallStatus(checks []DoctorCheckDTO) string {
	hasFail := false
	hasWarn := false
	for _, c := range checks {
		switch c.Status {
		case "fail":
			hasFail = true
		case "warn":
			hasWarn = true
		}
	}
	if hasFail {
		return "fail"
	}
	if hasWarn {
		return "warn"
	}
	return "pass"
}
