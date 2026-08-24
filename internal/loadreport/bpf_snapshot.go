package loadreport

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const bpfRegressionPct = 12.0

var ErrBPFGateCompareFailed = errors.New("loadreport: bpf gate baseline compare failed")

type BPFSnapshot struct {
	GeneratedAt string             `json:"generated_at"`
	SessionDir  string             `json:"session_dir,omitempty"`
	Metrics     map[string]float64 `json:"metrics"`
}

type BPFCompareRow struct {
	Metric   string
	Baseline string
	Current  string
	DeltaPct string
	Status   string
}

type BPFCompareResult struct {
	Rows []BPFCompareRow
	Pass bool
}

func CaptureBPFSnapshot(sessionDir string, summary *bpfSummary) BPFSnapshot {
	metrics := make(map[string]float64)

	if p99, ok := markerMaxP99(summary.Markers, "filter_check"); ok {
		metrics["filter_check_uprobe_p99_us"] = p99
	}
	if p99, ok := markerMaxP99(summary.Markers, "process_track"); ok {
		metrics["process_track_uprobe_p99_us"] = p99
	}
	if wall, ok := syscallWallPct(summary.Syscalls, "tracker", "epoll_wait"); ok {
		metrics["tracker_epoll_wait_wall_pct"] = wall
	}
	if rate, ok := hotSyscallRatePerSec(summary.HotSyscalls, "tracker", "futex", summary.DurationSec); ok {
		metrics["tracker_futex_per_sec"] = rate
	}
	if ratio, ok := ctxSwitchRatio(summary.PIDStats, "tracker"); ok {
		metrics["tracker_involuntary_ctx_ratio"] = ratio
	}
	if throttle, ok := maxCgroupThrottle(summary.CgroupSamples); ok {
		metrics["cgroup_cpu_throttle_pct_max"] = throttle
	}
	if pct, ok := loadgenOnCPUPct(summary.PIDStats); ok {
		metrics["loadgen_oncpu_pct"] = pct
	}
	metrics["tracker_outbound_connect"] = float64(trackerOutboundConnects(summary.Network))
	procPGConnects, _, procPGSendtoBytes := pgWireStats(summary.Network, "processor")
	metrics["processor_pg_connects"] = float64(procPGConnects)
	metrics["processor_pg_sendto_bytes"] = float64(procPGSendtoBytes)
	metrics["tracker_hw_cache_misses"] = float64(trackerHWCacheMisses(summary))
	metrics["max_fd_delta"] = float64(maxProcSampleInt64(summary.ProcSamples, func(p *procSample) int64 { return p.FDDelta }))
	metrics["max_thread_delta"] = float64(maxProcSampleInt64(summary.ProcSamples, func(p *procSample) int64 { return p.ThreadDelta }))
	metrics["max_peak_open_fds"] = float64(maxProcSampleInt64(summary.ProcSamples, func(p *procSample) int64 { return p.PeakOpenFDs }))
	metrics["max_rss_delta_kb"] = float64(maxProcSampleInt64(summary.ProcSamples, func(p *procSample) int64 { return p.RSSDelta }))
	metrics["max_peak_rss_kb"] = float64(maxProcSampleInt64(summary.ProcSamples, func(p *procSample) int64 { return p.PeakRSSKB }))
	metrics["max_majflt"] = float64(maxProcSampleInt64(summary.ProcSamples, func(p *procSample) int64 { return p.MajFlt }))

	for _, s := range summary.CgroupSamples {
		if s.MemoryMaxEvents > 0 {
			metrics["cgroup_memory_max_events"] = float64(s.MemoryMaxEvents)
			break
		}
	}

	return BPFSnapshot{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		SessionDir:  sessionDir,
		Metrics:     metrics,
	}
}

func maxProcSampleInt64(samples []procSample, pick func(*procSample) int64) int64 {
	var peak int64
	for i := range samples {
		v := pick(&samples[i])
		if v > peak {
			peak = v
		}
	}
	return peak
}

func LoadBPFSnapshot(path string) (BPFSnapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return BPFSnapshot{}, err
	}
	var snap BPFSnapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return BPFSnapshot{}, err
	}
	if snap.Metrics == nil {
		snap.Metrics = make(map[string]float64)
	}
	return snap, nil
}

func SaveBPFSnapshot(path string, snap BPFSnapshot) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func SeedBPFSnapshotBaseline(baselineDir, sessionDir string) error {
	summary, err := loadBPFSummary(sessionDir)
	if err != nil {
		return err
	}
	snap := CaptureBPFSnapshot(sessionDir, summary)
	if err := SaveBPFSnapshot(filepath.Join(baselineDir, "summary_snapshot.json"), snap); err != nil {
		return err
	}
	src := filepath.Join(sessionDir, "bpf", "maps", "summary.json")
	dst := filepath.Join(baselineDir, "summary.json")
	in, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, in, 0o644)
}

func CompareBPFSnapshots(baseline, current BPFSnapshot) BPFCompareResult {
	keys := make(map[string]struct{})
	for k := range baseline.Metrics {
		keys[k] = struct{}{}
	}
	for k := range current.Metrics {
		keys[k] = struct{}{}
	}
	sorted := make([]string, 0, len(keys))
	for k := range keys {
		sorted = append(sorted, k)
	}
	sort.Strings(sorted)

	var rows []BPFCompareRow
	pass := true
	for _, name := range sorted {
		b, bOK := baseline.Metrics[name]
		c, cOK := current.Metrics[name]
		row := BPFCompareRow{Metric: name}
		if !bOK {
			row.Baseline = "na"
			row.Current = formatFloat(c, 3)
			row.DeltaPct = "na"
			row.Status = "SKIP (new metric)"
			rows = append(rows, row)
			continue
		}
		if !cOK {
			row.Baseline = formatFloat(b, 3)
			row.Current = "na"
			row.DeltaPct = "na"
			row.Status = "SKIP (missing)"
			rows = append(rows, row)
			continue
		}
		row.Baseline = formatFloat(b, 3)
		row.Current = formatFloat(c, 3)
		row.DeltaPct = formatDeltaPct(b, c)
		row.Status, pass = compareBPFMetric(name, b, c, pass)
		rows = append(rows, row)
	}
	return BPFCompareResult{Rows: rows, Pass: pass}
}

func compareBPFMetric(name string, baseline, current float64, pass bool) (string, bool) {
	switch name {
	case "tracker_outbound_connect":
		if current > 0 {
			return "FAIL (connect leak)", false
		}
		return "OK", pass
	case "processor_pg_connects":
		if current > baseline {
			return "FAIL (PG connect churn)", false
		}
		return "OK", pass
	case "max_fd_delta", "max_thread_delta", "cgroup_memory_max_events", "max_rss_delta_kb", "max_peak_rss_kb", "max_majflt":
		if current > baseline {
			return "FAIL (regression)", false
		}
		return "OK", pass
	default:
		if baseline <= 0 {
			if current > baseline {
				return "FAIL (regression)", false
			}
			return "OK", pass
		}
		if current > baseline*(1+bpfRegressionPct/100) {
			return fmt.Sprintf("FAIL (>+%.0f%%)", bpfRegressionPct), false
		}
		return "OK", pass
	}
}

func formatDeltaPct(baseline, current float64) string {
	if baseline == 0 {
		if current == 0 {
			return "~"
		}
		return "+inf"
	}
	delta := (current - baseline) / baseline * 100
	if math.Abs(delta) < 0.05 {
		return "~"
	}
	return fmt.Sprintf("%+.1f%%", delta)
}

func WriteBPFGateCompareReport(baselineDir, sessionDir, promURL string) (string, error) {
	gatePath, err := WriteBPFGateReport(sessionDir, promURL)
	if err != nil && !errors.Is(err, ErrBPFGateFailed) {
		return "", err
	}
	_ = gatePath

	baselineSnapPath := filepath.Join(baselineDir, "summary_snapshot.json")
	currentSummary, err := loadBPFSummary(sessionDir)
	if err != nil {
		return "", err
	}
	currentSnap := CaptureBPFSnapshot(sessionDir, currentSummary)

	var compare BPFCompareResult
	if _, err := os.Stat(baselineSnapPath); err != nil {
		if os.IsNotExist(err) {
			if err := SeedBPFSnapshotBaseline(baselineDir, sessionDir); err != nil {
				return "", err
			}
			compare = BPFCompareResult{
				Rows: []BPFCompareRow{{
					Metric: "baseline",
					Status: "SEED (first run)",
				}},
				Pass: true,
			}
		} else {
			return "", err
		}
	} else {
		baselineSnap, err := LoadBPFSnapshot(baselineSnapPath)
		if err != nil {
			return "", err
		}
		compare = CompareBPFSnapshots(baselineSnap, currentSnap)
	}

	reportPath := filepath.Join(sessionDir, "bpf-gate-compare.md")
	var b strings.Builder
	b.WriteString("# BPF gate baseline compare\n\n")
	b.WriteString(fmt.Sprintf("Generated: %s\n", time.Now().UTC().Format(time.RFC3339)))
	b.WriteString(fmt.Sprintf("Baseline dir: `%s`\n", baselineDir))
	b.WriteString(fmt.Sprintf("Session: `%s`\n\n", sessionDir))
	b.WriteString("| Metric | Baseline | Current | Delta | Status |\n")
	b.WriteString("|--------|----------|---------|-------|--------|\n")
	for _, row := range compare.Rows {
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n",
			row.Metric, row.Baseline, row.Current, row.DeltaPct, row.Status)
	}
	if compare.Pass {
		b.WriteString("\n**Result: PASS**\n")
	} else {
		b.WriteString(fmt.Sprintf("\n**Result: FAIL** — regression > %.0f%% or cold-path leak (.cursor/rules/ci.mdc#bpf-ci-arch).\n", bpfRegressionPct))
	}
	if err := os.WriteFile(reportPath, []byte(b.String()), 0o644); err != nil {
		return "", err
	}
	if !compare.Pass {
		return reportPath, fmt.Errorf("%w: see %s", ErrBPFGateCompareFailed, reportPath)
	}
	return reportPath, nil
}

func bpfColdGateEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("BPF_COLD_GATE"))) {
	case "1", "true", "yes":
		return true
	default:
		return false
	}
}

func checkBPFColdPathChecks(summary *bpfSummary) []BPFGateCheck {
	maxFD := maxProcSampleInt64(summary.ProcSamples, func(p *procSample) int64 { return p.FDDelta })
	maxThread := maxProcSampleInt64(summary.ProcSamples, func(p *procSample) int64 { return p.ThreadDelta })

	checks := []BPFGateCheck{
		{
			Name:   "max_fd_delta",
			Value:  strconvFormatInt(maxFD),
			Limit:  "0",
			OK:     maxFD <= 0,
			Detail: "cold path: FD delta after soak",
		},
		{
			Name:   "max_thread_delta",
			Value:  strconvFormatInt(maxThread),
			Limit:  "50",
			OK:     maxThread <= 50,
			Detail: "cold path: thread/goroutine delta",
		},
	}
	for i := range summary.ProcSamples {
		s := &summary.ProcSamples[i]
		if s.FDOpenPerSec > 0 && s.FDClosePerSec > 0 && s.FDOpenPerSec > s.FDClosePerSec*2 {
			checks = append(checks, BPFGateCheck{
				Name:   "fd_open_close_imbalance_" + s.Role,
				Value:  formatFloat(s.FDOpenPerSec, 1),
				Limit:  formatFloat(s.FDClosePerSec*2, 1),
				OK:     false,
				Detail: "open_rate >> close_rate (dangling connections)",
			})
			break
		}
	}
	checks = append(checks, checkBPFRSSChecks(summary, []string{"control", "processor", "region-proxy"}, 51200)...)
	return checks
}

func strconvFormatInt(v int64) string {
	return fmt.Sprintf("%d", v)
}
