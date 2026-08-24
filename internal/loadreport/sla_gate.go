package loadreport

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	OpenRTBExchangeP99MsLimit = 80.0
	TrackerHandlerP99MsLimit  = 80.0
)

type SLACheck struct {
	Name    string
	ValueMs float64
	LimitMs float64
	OK      bool
}

type SLAResult struct {
	Checks []SLACheck
	Pass   bool
}

func CheckOpenRTBSLA(promURL string) (SLAResult, error) {
	prom := newPromClient(promURL)
	checks := []SLACheck{
		checkScalarMS(prom, "openrtb_exchange_p99",
			`histogram_quantile(0.99, sum(rate(ad_rtb_exchange_duration_seconds_bucket{job="tracker"}[5m])) by (le)) * 1000`,
			OpenRTBExchangeP99MsLimit),
		checkScalarMS(prom, "tracker_handler_p99",
			`histogram_quantile(0.99, sum(rate(ad_http_request_duration_seconds_bucket{job="tracker"}[5m])) by (le)) * 1000`,
			TrackerHandlerP99MsLimit),
	}
	pass := true
	for i := range checks {
		if !checks[i].OK {
			pass = false
		}
	}
	return SLAResult{Checks: checks, Pass: pass}, nil
}

func checkScalarMS(prom *promClient, name, query string, limit float64) SLACheck {
	raw := prom.scalar(query)
	chk := SLACheck{Name: name, LimitMs: limit}
	if raw == "na" || raw == "" {
		chk.OK = true
		return chk
	}
	val, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil {
		chk.OK = true
		return chk
	}
	chk.ValueMs = val
	chk.OK = val < limit
	return chk
}

func WriteSLAReport(outDir, promURL string) (string, error) {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", err
	}
	result, err := CheckOpenRTBSLA(promURL)
	if err != nil {
		return "", err
	}
	path := filepath.Join(outDir, "sla-gate.md")
	var b strings.Builder
	b.WriteString("# OpenRTB 2.6 SLA gate (R4)\n\n")
	b.WriteString(fmt.Sprintf("Generated: %s\n", time.Now().UTC().Format(time.RFC3339)))
	b.WriteString(fmt.Sprintf("Prometheus: %s\n\n", promURL))
	b.WriteString("| Check | Value (ms) | Limit (ms) | Status |\n")
	b.WriteString("|-------|------------|------------|--------|\n")
	for _, c := range result.Checks {
		val := "na"
		if c.ValueMs > 0 {
			val = fmt.Sprintf("%.2f", c.ValueMs)
		}
		status := "PASS"
		if !c.OK {
			status = "FAIL"
		}
		fmt.Fprintf(&b, "| %s | %s | %.0f | %s |\n", c.Name, val, c.LimitMs, status)
	}
	if result.Pass {
		b.WriteString("\n**Result: PASS**\n")
	} else {
		b.WriteString("\n**Result: FAIL** - exchange or tracker p99 exceeded section 5 budget.\n")
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return "", err
	}
	if !result.Pass {
		return path, fmt.Errorf("SLA gate failed: see %s", path)
	}
	return path, nil
}
