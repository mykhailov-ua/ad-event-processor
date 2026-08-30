//go:build !linux

// Non-linux stub: hardware_perf omitted from bpf/maps/summary.json.
package main

func (r *probeRun) collectHardwarePerf(pidStats []dumpedPIDStats) []map[string]any {
	return nil
}
