package loadreport

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type StrictContentionSnapshot struct {
	SessionDir string
	PromURL    string
	Generated  string

	TrackerP99Ms              string
	TrackerP95Ms              string
	TrackerRPS                string
	SyncLagMaxSec             string
	SyncLagTop                []promVectorRow
	LocalQuotaBlockPerSec     string
	ProcessorWriteAcquireP99S string
	RedisLuaP99MaxMs          string
	RedisLuaP99ByShard        []promVectorRow
}

func promRateWindow() string {
	if w := strings.TrimSpace(os.Getenv("LOAD_REPORT_RATE_WINDOW")); w != "" {
		return w
	}
	return "5m"
}

func CaptureStrictContention(sessionDir, promURL string) (*StrictContentionSnapshot, error) {
	prom := newPromClient(promURL)
	rateWindow := promRateWindow()
	rate := func(q string) string {
		return strings.ReplaceAll(q, "${window}", rateWindow)
	}
	snap := &StrictContentionSnapshot{
		SessionDir: sessionDir,
		PromURL:    promURL,
		Generated:  time.Now().UTC().Format(time.RFC3339Nano),
		TrackerP99Ms: prom.scalar(rate(
			`histogram_quantile(0.99, sum(rate(ad_http_request_duration_seconds_bucket{job="tracker"}[${window}])) by (le)) * 1000`,
		)),
		TrackerP95Ms: prom.scalar(rate(
			`histogram_quantile(0.95, sum(rate(ad_http_request_duration_seconds_bucket{job="tracker"}[${window}])) by (le)) * 1000`,
		)),
		TrackerRPS:            prom.scalar(rate(`sum(rate(ad_http_request_duration_seconds_count{job="tracker"}[${window}]))`)),
		SyncLagMaxSec:         prom.scalar(`max(ad_sync_lag_seconds)`),
		LocalQuotaBlockPerSec: prom.scalar(rate(`sum(rate(ad_tracker_local_quota_block_total{job="tracker"}[${window}]))`)),
		ProcessorWriteAcquireP99S: prom.scalar(rate(
			`histogram_quantile(0.99, sum(rate(ad_processor_write_acquire_wait_seconds_bucket{job="processor"}[${window}])) by (le))`,
		)),
	}
	snap.SyncLagTop, _ = prom.vector(`topk(5, ad_sync_lag_seconds)`)
	shards, _ := prom.vector(rate(`sum(rate(ad_redis_ops_total{job="tracker"}[${window}])) by (shard)`))
	var luaRows []promVectorRow
	maxP99 := 0.0
	for _, row := range shards {
		if row.Labels == "" {
			continue
		}
		shard := strings.TrimPrefix(row.Labels, "shard=")
		p99Str := prom.scalar(fmt.Sprintf(
			`histogram_quantile(0.99, sum(rate(ad_redis_lua_duration_seconds_bucket{job="tracker",shard="%s"}[%s])) by (le)) * 1000`,
			shard, rateWindow,
		))
		luaRows = append(luaRows, promVectorRow{Value: p99Str, Labels: row.Labels})
		if p99, err := strconv.ParseFloat(p99Str, 64); err == nil && p99 > maxP99 {
			maxP99 = p99
		}
	}
	sort.Slice(luaRows, func(i, j int) bool {
		vi, _ := strconv.ParseFloat(luaRows[i].Value, 64)
		vj, _ := strconv.ParseFloat(luaRows[j].Value, 64)
		return vi > vj
	})
	snap.RedisLuaP99ByShard = luaRows
	if maxP99 > 0 {
		snap.RedisLuaP99MaxMs = fmt.Sprintf("%.3f", maxP99)
	} else {
		snap.RedisLuaP99MaxMs = prom.scalar(rate(
			`histogram_quantile(0.99, sum(rate(ad_redis_lua_duration_seconds_bucket{job="tracker"}[${window}])) by (le)) * 1000`,
		))
	}
	return snap, nil
}

func WriteStrictContentionReport(sessionDir, promURL string) (string, error) {
	snap, err := CaptureStrictContention(sessionDir, promURL)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(sessionDir, "strict-contention.md")
	if err := os.WriteFile(path, []byte(formatStrictContentionMD(snap, "")), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func WriteStrictContentionCompare(baselineDir, treatmentDir, promURL string) (string, error) {
	base, err := CaptureStrictContention(baselineDir, promURL)
	if err != nil {
		return "", err
	}
	treat, err := CaptureStrictContention(treatmentDir, promURL)
	if err != nil {
		return "", err
	}
	outDir := filepath.Dir(baselineDir)
	path := filepath.Join(outDir, "strict-contention-compare.md")
	body := formatStrictContentionCompareMD(base, treat)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func formatStrictContentionMD(s *StrictContentionSnapshot, title string) string {
	if title == "" {
		title = "StrictFlush / settlement contention"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", title)
	fmt.Fprintf(&b, "Generated: %s\n", s.Generated)
	fmt.Fprintf(&b, "Session: `%s`\n", s.SessionDir)
	fmt.Fprintf(&b, "Prometheus: %s\n", s.PromURL)
	fmt.Fprintf(&b, "Rate window: `%s`\n\n", promRateWindow())

	b.WriteString("## Thresholds (reference)\n\n")
	b.WriteString("| Knob | Default | Gate |\n")
	b.WriteString("|------|---------|------|\n")
	b.WriteString("| `strictThresholdMicro` / `QUOTA_STRICT_THRESHOLD_MICRO` | 5_000_000 | StrictFlush + ingest strict enter |\n")
	b.WriteString("| `QUOTA_STRICT_EXIT_MICRO` | 8_000_000 | ingest strict exit (hysteresis) |\n")
	b.WriteString("| Tracker p99 SLA | < 80 ms | `ad_http_request_duration_seconds` |\n")
	b.WriteString("| Redis Lua p99 SLA | < 10 ms/shard | `ad_redis_lua_duration_seconds` |\n\n")

	b.WriteString("## Metrics\n\n")
	b.WriteString("| Metric | Value |\n")
	b.WriteString("|--------|-------|\n")
	fmt.Fprintf(&b, "| Tracker RPS | %s |\n", s.TrackerRPS)
	fmt.Fprintf(&b, "| Tracker p95 (ms) | %s |\n", s.TrackerP95Ms)
	fmt.Fprintf(&b, "| Tracker p99 (ms) | %s |\n", s.TrackerP99Ms)
	fmt.Fprintf(&b, "| `ad_sync_lag_seconds` max | %s |\n", s.SyncLagMaxSec)
	fmt.Fprintf(&b, "| `ad_tracker_local_quota_block_total` rate/s | %s |\n", s.LocalQuotaBlockPerSec)
	fmt.Fprintf(&b, "| `ad_processor_write_acquire_wait_seconds` p99 | %s |\n", s.ProcessorWriteAcquireP99S)
	fmt.Fprintf(&b, "| Redis Lua p99 max (ms) | %s |\n", s.RedisLuaP99MaxMs)
	b.WriteString("\n")

	if len(s.SyncLagTop) > 0 {
		b.WriteString("### Top `ad_sync_lag_seconds`\n\n")
		for _, row := range s.SyncLagTop {
			fmt.Fprintf(&b, "- %s → %s sec\n", row.Labels, row.Value)
		}
		b.WriteString("\n")
	}
	if len(s.RedisLuaP99ByShard) > 0 {
		b.WriteString("### Redis Lua p99 by shard (ms)\n\n")
		for _, row := range s.RedisLuaP99ByShard {
			fmt.Fprintf(&b, "- %s → %s ms\n", row.Labels, row.Value)
		}
		b.WriteString("\n")
	}

	b.WriteString("## Interpretation\n\n")
	b.WriteString("- **Sync lag ↑ + local quota block ↑** under low remaining → strict-mode chain active (expected under low-budget drill).\n")
	b.WriteString("- **Tracker p99 ≥ 80 ms** or **Lua p99 ≥ 10 ms** sustained → perf regression; tuning thresholds alone is insufficient.\n")
	b.WriteString("- Tuning `QUOTA_STRICT_*` requires this report **before vs after** plus unchanged tracker p99.\n")
	return b.String()
}

func formatStrictContentionCompareMD(base, treat *StrictContentionSnapshot) string {
	var b strings.Builder
	b.WriteString("# StrictFlush contention — baseline vs low-budget\n\n")
	fmt.Fprintf(&b, "Generated: %s\n\n", time.Now().UTC().Format(time.RFC3339Nano))
	fmt.Fprintf(&b, "Baseline session: `%s`\n", base.SessionDir)
	fmt.Fprintf(&b, "Low-budget session: `%s`\n\n", treat.SessionDir)

	writeCompareRow := func(name string, a, c string) {
		fmt.Fprintf(&b, "| %s | %s | %s | %s |\n", name, a, c, deltaLabel(a, c))
	}

	b.WriteString("| Metric | Baseline | Low-budget | Δ |\n")
	b.WriteString("|--------|----------|------------|---|\n")
	writeCompareRow("Tracker p99 (ms)", base.TrackerP99Ms, treat.TrackerP99Ms)
	writeCompareRow("Tracker RPS", base.TrackerRPS, treat.TrackerRPS)
	writeCompareRow("sync_lag max (s)", base.SyncLagMaxSec, treat.SyncLagMaxSec)
	writeCompareRow("local_quota_block/s", base.LocalQuotaBlockPerSec, treat.LocalQuotaBlockPerSec)
	writeCompareRow("processor write acquire p99 (s)", base.ProcessorWriteAcquireP99S, treat.ProcessorWriteAcquireP99S)
	writeCompareRow("Redis Lua p99 max (ms)", base.RedisLuaP99MaxMs, treat.RedisLuaP99MaxMs)
	b.WriteString("\n")

	b.WriteString("## Verdict (automated heuristics)\n\n")
	verdicts := strictContentionVerdicts(base, treat)
	for _, v := range verdicts {
		b.WriteString("- " + v + "\n")
	}
	b.WriteString("\n")
	b.WriteString("## Baseline detail\n\n")
	b.WriteString(formatStrictContentionMD(base, "Baseline"))
	b.WriteString("\n## Low-budget detail\n\n")
	b.WriteString(formatStrictContentionMD(treat, "Low-budget"))
	return b.String()
}

func deltaLabel(a, c string) string {
	af, errA := strconv.ParseFloat(a, 64)
	cf, errC := strconv.ParseFloat(c, 64)
	if errA != nil || errC != nil || a == "na" || c == "na" {
		return "n/a"
	}
	d := cf - af
	if d > 0 {
		return fmt.Sprintf("+%.3f", d)
	}
	return fmt.Sprintf("%.3f", d)
}

func strictContentionVerdicts(base, treat *StrictContentionSnapshot) []string {
	var out []string
	if treatP99, err := strconv.ParseFloat(treat.TrackerP99Ms, 64); err == nil && treatP99 >= 80 {
		out = append(out, fmt.Sprintf("FAIL: tracker p99 %.1f ms ≥ 80 ms SLA (harness=loadgen constrained stack)", treatP99))
	} else {
		out = append(out, "PASS: tracker p99 < 80 ms (or na)")
	}
	if lagB, errB := strconv.ParseFloat(base.SyncLagMaxSec, 64); errB == nil {
		if lagT, errT := strconv.ParseFloat(treat.SyncLagMaxSec, 64); errT == nil && lagT > lagB+1 {
			out = append(out, fmt.Sprintf("SIGNAL: sync_lag max rose %.3f → %.3f (strict chain likely)", lagB, lagT))
		}
	}
	if blockT, err := strconv.ParseFloat(treat.LocalQuotaBlockPerSec, 64); err == nil && blockT > 0 {
		out = append(out, fmt.Sprintf("SIGNAL: local_quota_block rate %.3f/s > 0 under low-budget drill", blockT))
	}
	if luaT, err := strconv.ParseFloat(treat.RedisLuaP99MaxMs, 64); err == nil && luaT >= 10 {
		out = append(out, fmt.Sprintf("WARN: Redis Lua p99 max %.1f ms ≥ 10 ms/shard SLA", luaT))
	}
	if len(out) == 0 {
		out = append(out, "No automated signals (metrics na or drill too short)")
	}
	return out
}
