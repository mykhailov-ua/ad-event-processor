package doctor

import (
	"context"
	"strings"
	"time"

	"ad-event-processor/internal/config"
)

type Options struct {
	Only    []string
	Timeout time.Duration
	Probes  []Probe
}

func DefaultProbes(deps ProbeDeps) []Probe {
	deps = WithCLILicenseDeps(deps)
	probes := []Probe{
		KernelProbe{},
		SysctlProbe{},
		ListenBacklogProbe{},
		RedisProbe{Deps: deps},
		SlotMapProbe{Deps: deps},
		ClickHouseProbe{Deps: deps},
		DiskProbe{},
		TLSProbe{Deps: deps},
	}
	if config.LicenseProbeEnabled() {
		probes = append(probes, licenseProbe(deps))
	}
	return probes
}

func Run(ctx context.Context, opts Options) Report {
	if opts.Timeout <= 0 {
		opts.Timeout = 60 * time.Second
	}
	runCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	filter := newProbeFilter(opts.Only)
	var results []Result
	for _, probe := range opts.Probes {
		if probe == nil {
			continue
		}
		if !filter.allow(probe.Name()) {
			continue
		}
		results = append(results, probe.Run(runCtx))
	}
	return Report{Results: results}
}

type probeFilter struct {
	allowed map[string]struct{}
	active  bool
}

func newProbeFilter(only []string) probeFilter {
	if len(only) == 0 {
		return probeFilter{}
	}
	allowed := make(map[string]struct{}, len(only))
	for _, name := range only {
		name = strings.TrimSpace(strings.ToLower(name))
		if name == "" {
			continue
		}
		allowed[name] = struct{}{}
	}
	return probeFilter{allowed: allowed, active: len(allowed) > 0}
}

func (f probeFilter) allow(name string) bool {
	if !f.active {
		return true
	}
	_, ok := f.allowed[strings.ToLower(name)]
	return ok
}
