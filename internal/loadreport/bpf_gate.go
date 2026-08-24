package loadreport

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	bpfFilterCheckP99FailUs    = 1000.0
	bpfProcessTrackP99FailUs   = 5000.0
	bpfEpollWaitWallFailPct    = 60.0
	bpfFutexPerSecFail         = 50000.0
	bpfCtxRatioFail            = 5.0
	bpfThrottleFailPct         = 15.0
	bpfLoadgenOnCPUFailPct     = 25.0
	bpfRedisLuaP99FailMs       = 10.0
	bpfTrackerHandlerP99FailMs = 80.0
)

var ErrBPFGateFailed = errors.New("loadreport: bpf resource gate failed")

type BPFGateCheck struct {
	Name   string
	Value  string
	Limit  string
	OK     bool
	Detail string
}

type BPFGateResult struct {
	Checks []BPFGateCheck
	Pass   bool
}

func bpfGateStrict() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("BPF_GATE_STRICT"))) {
	case "1", "true", "yes":
		return true
	default:
		return false
	}
}

func loadBPFSummary(outDir string) (*bpfSummary, error) {
	summaryPath := filepath.Join(outDir, "bpf", "maps", "summary.json")
	data, err := os.ReadFile(summaryPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNoBPFSummary
		}
		return nil, err
	}
	var summary bpfSummary
	if err := json.Unmarshal(data, &summary); err != nil {
		return nil, err
	}
	return &summary, nil
}

func CheckBPFResourceGate(outDir, promURL string) (BPFGateResult, error) {
	summary, err := loadBPFSummary(outDir)
	if err != nil {
		if errors.Is(err, ErrNoBPFSummary) {
			if bpfGateStrict() {
				return BPFGateResult{}, err
			}
			return BPFGateResult{
				Checks: []BPFGateCheck{{
					Name:   "bpf_summary",
					Value:  "missing",
					Limit:  "present",
					OK:     true,
					Detail: "skipped (no BPF session; set AD_EVENT_PROCESSOR_BPF_PROBE=1)",
				}},
				Pass: true,
			}, nil
		}
		return BPFGateResult{}, err
	}

	checks := append([]BPFGateCheck(nil), checkBPFSummaryChecks(summary)...)
	if bpfColdGateEnabled() {
		checks = append(checks, checkBPFColdPathChecks(summary)...)
		checks = append(checks, checkBPFPGWireChecks(summary)...)
	}
	promChecks := checkBPFGatePrometheus(promURL)
	checks = append(checks, promChecks...)

	pass := true
	for _, c := range checks {
		if !c.OK {
			pass = false
		}
	}
	return BPFGateResult{Checks: checks, Pass: pass}, nil
}

func checkBPFSummaryChecks(summary *bpfSummary) []BPFGateCheck {
	var checks []BPFGateCheck

	if p99, ok := markerMaxP99(summary.Markers, "filter_check"); ok {
		checks = append(checks, BPFGateCheck{
			Name:   "filter_check_uprobe_p99_us",
			Value:  formatFloat(p99, 1),
			Limit:  formatFloat(bpfFilterCheckP99FailUs, 1),
			OK:     p99 < bpfFilterCheckP99FailUs,
			Detail: "FilterEngine.Check uprobe p99",
		})
	} else {
		checks = append(checks, labAwareMissingCheck(
			"filter_check_uprobe_p99_us",
			formatFloat(bpfFilterCheckP99FailUs, 1),
			"no filter_check uprobes; build tracker with bpftrace tag",
		))
	}

	if p99, ok := markerMaxP99(summary.Markers, "process_track"); ok {
		checks = append(checks, BPFGateCheck{
			Name:   "process_track_uprobe_p99_us",
			Value:  formatFloat(p99, 1),
			Limit:  formatFloat(bpfProcessTrackP99FailUs, 1),
			OK:     p99 < bpfProcessTrackP99FailUs,
			Detail: "/track handler uprobe p99",
		})
	} else {
		checks = append(checks, labAwareMissingCheck(
			"process_track_uprobe_p99_us",
			formatFloat(bpfProcessTrackP99FailUs, 1),
			"no process_track uprobes; build tracker with bpftrace tag",
		))
	}

	if wall, ok := syscallWallPct(summary.Syscalls, "tracker", "epoll_wait"); ok {
		checks = append(checks, BPFGateCheck{
			Name:   "tracker_epoll_wait_wall_pct",
			Value:  formatFloat(wall, 1),
			Limit:  formatFloat(bpfEpollWaitWallFailPct, 1),
			OK:     wall < bpfEpollWaitWallFailPct,
			Detail: "tracker epoll_wait wall time share",
		})
	} else {
		checks = append(checks, skipCheck("tracker_epoll_wait_wall_pct", "skipped (no tracker epoll_wait samples)"))
	}

	if rate, ok := hotSyscallRatePerSec(summary.HotSyscalls, "tracker", "futex", summary.DurationSec); ok {
		if bpfGateLabProfile() {
			checks = append(checks, labSkipCheck(
				"tracker_futex_per_sec",
				formatFloat(rate, 1),
				"Go scheduler futex rate not comparable on constrained Docker lab",
			))
		} else {
			checks = append(checks, BPFGateCheck{
				Name:   "tracker_futex_per_sec",
				Value:  formatFloat(rate, 1),
				Limit:  formatFloat(bpfFutexPerSecFail, 1),
				OK:     rate < bpfFutexPerSecFail,
				Detail: "futex syscall rate (lock contention)",
			})
		}
	} else {
		checks = append(checks, skipCheck("tracker_futex_per_sec", "skipped (no tracker futex samples)"))
	}

	if ratio, ok := ctxSwitchRatio(summary.PIDStats, "tracker"); ok {
		checks = append(checks, BPFGateCheck{
			Name:   "tracker_involuntary_ctx_ratio",
			Value:  formatFloat(ratio, 2),
			Limit:  formatFloat(bpfCtxRatioFail, 2),
			OK:     ratio < bpfCtxRatioFail,
			Detail: "involuntary / voluntary context switches",
		})
	} else {
		checks = append(checks, skipCheck("tracker_involuntary_ctx_ratio", "skipped (no tracker ctx switch samples)"))
	}

	if throttle, ok := maxCgroupThrottle(summary.CgroupSamples); ok {
		checks = append(checks, BPFGateCheck{
			Name:   "cgroup_cpu_throttle_pct_max",
			Value:  formatFloat(throttle, 1),
			Limit:  formatFloat(bpfThrottleFailPct, 1),
			OK:     throttle < bpfThrottleFailPct,
			Detail: "max cgroup CPU throttle across targets",
		})
	} else {
		checks = append(checks, skipCheck("cgroup_cpu_throttle_pct_max", "skipped (no cgroup samples)"))
	}

	if pct, ok := loadgenOnCPUPct(summary.PIDStats); ok {
		checks = append(checks, BPFGateCheck{
			Name:   "loadgen_oncpu_pct",
			Value:  formatFloat(pct, 1),
			Limit:  formatFloat(bpfLoadgenOnCPUFailPct, 1),
			OK:     pct < bpfLoadgenOnCPUFailPct,
			Detail: "load generator on-CPU share vs tracked processes",
		})
	} else {
		checks = append(checks, skipCheck("loadgen_oncpu_pct", "skipped (no loadgen or tracker on-CPU samples)"))
	}

	connects := trackerOutboundConnects(summary.Network)
	if bpfGateLabProfile() {
		checks = append(checks, labSkipCheck(
			"tracker_outbound_connect",
			strconv.FormatInt(connects, 10),
			"load-test uses TCP Redis; UDS appliance has connect()==0",
		))
	} else {
		checks = append(checks, BPFGateCheck{
			Name:   "tracker_outbound_connect",
			Value:  strconv.FormatInt(connects, 10),
			Limit:  "0",
			OK:     connects == 0,
			Detail: "tracker must not call connect() on hot path (T9)",
		})
	}

	checks = append(checks, checkBPFFDLeak(summary)...)
	if !bpfGateLabProfile() {
		checks = append(checks, checkBPFRSSChecks(summary, []string{"tracker"}, 5120)...)
	}

	for _, s := range summary.CgroupSamples {
		if s.MemoryMaxEvents > 0 {
			checks = append(checks, BPFGateCheck{
				Name:   "cgroup_memory_max_events",
				Value:  strconv.FormatInt(s.MemoryMaxEvents, 10),
				Limit:  "0",
				OK:     false,
				Detail: fmt.Sprintf("container %s hit memory.max", s.Role),
			})
			break
		}
	}

	return checks
}

func checkBPFGatePrometheus(promURL string) []BPFGateCheck {
	prom := newPromClient(promURL)
	rateWindow := promRateWindow()

	handlerRaw := prom.scalar(strings.ReplaceAll(
		`histogram_quantile(0.99, sum(rate(ad_http_request_duration_seconds_bucket{job="tracker"}[${window}])) by (le)) * 1000`,
		"${window}", rateWindow,
	))
	handlerCheck := promScalarCheck(
		"tracker_handler_p99_ms",
		formatFloat(bpfTrackerHandlerP99FailMs, 0),
		handlerRaw,
		"platform SLA handler p99",
	)

	luaMax, skipped := redisLuaP99MaxMs(prom, rateWindow)
	luaCheck := BPFGateCheck{
		Name:   "redis_lua_p99_max_ms",
		Limit:  formatFloat(bpfRedisLuaP99FailMs, 0),
		Detail: "unified-filter Lua p99 per shard",
	}
	if skipped {
		luaCheck.Value = "na"
		if bpfGateStrict() && !bpfGateLabProfile() {
			luaCheck.OK = false
			luaCheck.Detail = "required in strict mode (Prometheus unavailable)"
		} else {
			luaCheck.OK = true
			luaCheck.Detail = "skipped (Prometheus unavailable)"
		}
	} else {
		luaCheck.Value = formatFloat(luaMax, 3)
		luaCheck.OK = luaMax < bpfRedisLuaP99FailMs
	}

	var checks []BPFGateCheck
	if bpfGateLabProfile() && bpfColdGateEnabled() {
		checks = append(checks,
			labSkipCheck("tracker_handler_p99_ms", handlerCheck.Value, "lab cold soak; see PERF runner for hot SLA"),
			labSkipCheck("redis_lua_p99_max_ms", luaCheck.Value, "lab uses TCP Redis shards"),
		)
	} else {
		checks = append(checks, handlerCheck, luaCheck)
	}
	checks = append(checks, checkBPFDiskSpoolPrometheus(prom)...)
	checks = append(checks, checkBPFRedisPoolPrometheus(prom, rateWindow)...)
	checks = append(checks, checkBPFExtraGatesPrometheus(prom, rateWindow)...)
	return checks
}

func redisLuaP99MaxMs(prom *promClient, rateWindow string) (float64, bool) {
	rate := func(q string) string {
		return strings.ReplaceAll(q, "${window}", rateWindow)
	}
	shards, _ := prom.vector(rate(`sum(rate(ad_redis_ops_total{job="tracker"}[${window}])) by (shard)`))
	maxP99 := 0.0
	found := false
	for _, row := range shards {
		if row.Labels == "" {
			continue
		}
		shard := strings.TrimPrefix(row.Labels, "shard=")
		p99Str := prom.scalar(fmt.Sprintf(
			`histogram_quantile(0.99, sum(rate(ad_redis_lua_duration_seconds_bucket{job="tracker",shard=%q}[%s])) by (le)) * 1000`,
			shard, rateWindow,
		))
		if p99Str == "na" || p99Str == "" {
			continue
		}
		p99, err := strconv.ParseFloat(strings.TrimSpace(p99Str), 64)
		if err != nil {
			continue
		}
		found = true
		if p99 > maxP99 {
			maxP99 = p99
		}
	}
	if found {
		return maxP99, false
	}
	fallback := prom.scalar(rate(
		`histogram_quantile(0.99, sum(rate(ad_redis_lua_duration_seconds_bucket{job="tracker"}[${window}])) by (le)) * 1000`,
	))
	if fallback == "na" || fallback == "" {
		return 0, true
	}
	p99, err := strconv.ParseFloat(strings.TrimSpace(fallback), 64)
	if err != nil {
		return 0, true
	}
	return p99, false
}

func WriteBPFGateReport(outDir, promURL string) (string, error) {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", err
	}
	result, err := CheckBPFResourceGate(outDir, promURL)
	if err != nil {
		return "", err
	}
	path := filepath.Join(outDir, "bpf-gate.md")
	var b strings.Builder
	b.WriteString("# BPF resource gate (hot path)\n\n")
	b.WriteString(fmt.Sprintf("Generated: %s\n", time.Now().UTC().Format(time.RFC3339)))
	b.WriteString(fmt.Sprintf("Session: `%s`\n", outDir))
	b.WriteString(fmt.Sprintf("Prometheus: `%s`\n", promURL))
	b.WriteString(fmt.Sprintf("Strict: `%v`\n\n", bpfGateStrict()))
	b.WriteString("| Check | Value | Limit | Status | Detail |\n")
	b.WriteString("|-------|-------|-------|--------|--------|\n")
	for _, c := range result.Checks {
		status := "PASS"
		if !c.OK {
			status = "FAIL"
		} else if strings.HasPrefix(c.Detail, "skipped") {
			status = "SKIP"
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n", c.Name, c.Value, c.Limit, status, c.Detail)
	}
	if result.Pass {
		b.WriteString("\n**Result: PASS**\n")
	} else {
		b.WriteString("\n**Result: FAIL** - see .cursor/rules/ci.mdc#bpf-hot-gate thresholds.\n")
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return "", err
	}
	if !result.Pass {
		return path, fmt.Errorf("%w: see %s", ErrBPFGateFailed, path)
	}
	return path, nil
}

func skipCheck(name, detail string) BPFGateCheck {
	return BPFGateCheck{
		Name:   name,
		Value:  "na",
		Limit:  "n/a",
		OK:     true,
		Detail: detail,
	}
}

func strictMissingCheck(name, limit, detail string) BPFGateCheck {
	if bpfGateStrict() {
		return BPFGateCheck{
			Name:   name,
			Value:  "missing",
			Limit:  limit,
			OK:     false,
			Detail: detail,
		}
	}
	return skipCheck(name, "skipped ("+detail+")")
}

func promScalarCheck(name, limit, raw, failDetail string) BPFGateCheck {
	check := BPFGateCheck{
		Name:   name,
		Limit:  limit,
		Detail: failDetail,
	}
	if raw == "na" || raw == "" {
		check.Value = "na"
		if bpfGateStrict() {
			check.OK = false
			check.Detail = "required in strict mode (Prometheus unavailable)"
		} else {
			check.OK = true
			check.Detail = "skipped (Prometheus unavailable)"
		}
		return check
	}
	val, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil {
		check.Value = raw
		if bpfGateStrict() {
			check.OK = false
			check.Detail = "required in strict mode (unparseable Prometheus scalar)"
		} else {
			check.OK = true
			check.Detail = "skipped (unparseable scalar)"
		}
		return check
	}
	limitVal, _ := strconv.ParseFloat(limit, 64)
	check.Value = formatFloat(val, 2)
	check.OK = val < limitVal
	return check
}

func markerMaxP99(markers []markerStat, name string) (float64, bool) {
	var maxP99 float64
	found := false
	for _, m := range markers {
		if m.Marker != name {
			continue
		}
		found = true
		if m.P99Us > maxP99 {
			maxP99 = m.P99Us
		}
	}
	return maxP99, found
}

func syscallWallPct(syscalls []syscallStat, role, syscall string) (float64, bool) {
	for _, s := range syscalls {
		if s.Role == role && s.Syscall == syscall {
			return s.WallPct, true
		}
	}
	return 0, false
}

func hotSyscallRatePerSec(hot []syscallStat, role, syscall string, durationSec float64) (float64, bool) {
	if durationSec <= 0 {
		durationSec = 1
	}
	var count int64
	found := false
	for _, s := range hot {
		if s.Role == role && s.Syscall == syscall {
			count += s.Count
			found = true
		}
	}
	if !found {
		return 0, false
	}
	return float64(count) / durationSec, true
}

func ctxSwitchRatio(stats []pidStat, role string) (float64, bool) {
	for i := range stats {
		if stats[i].Role != role {
			continue
		}
		if stats[i].VoluntaryCtx <= 0 {
			return 0, false
		}
		return float64(stats[i].InvoluntaryCtx) / float64(stats[i].VoluntaryCtx), true
	}
	return 0, false
}

func maxCgroupThrottle(samples []cgroupSample) (float64, bool) {
	if len(samples) == 0 {
		return 0, false
	}
	maxPct := samples[0].ThrottlePct
	for i := 1; i < len(samples); i++ {
		if samples[i].ThrottlePct > maxPct {
			maxPct = samples[i].ThrottlePct
		}
	}
	return maxPct, true
}

func loadgenOnCPUPct(stats []pidStat) (float64, bool) {
	var totalOnCPU, loadgenOnCPU int64
	var hasLoadgen bool
	for i := range stats {
		totalOnCPU += stats[i].OnCPUNs
		if stats[i].Role == "loadgen" {
			hasLoadgen = true
			loadgenOnCPU += stats[i].OnCPUNs
		}
	}
	if !hasLoadgen || totalOnCPU <= 0 {
		return 0, false
	}
	return float64(loadgenOnCPU) / float64(totalOnCPU) * 100, true
}

func trackerOutboundConnects(network []networkStat) int64 {
	var connects int64
	for _, n := range network {
		if n.Role == "tracker" {
			connects += n.Connects
		}
	}
	return connects
}

func formatFloat(v float64, prec int) string {
	return strconv.FormatFloat(v, 'f', prec, 64)
}
