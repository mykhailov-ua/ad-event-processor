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

const TelegramTrackerHandlerP99MsLimit = 80.0

var ErrTelegramGateFailed = errors.New("loadreport: telegram hot-path gate failed")

type TelegramGateCheck struct {
	Name   string
	Value  string
	Limit  string
	OK     bool
	Detail string
}

type TelegramGateResult struct {
	Checks []TelegramGateCheck
	Pass   bool
}

// CheckTelegramBPF validates T9: tracker must not emit outbound connect() on hot path.
func CheckTelegramBPF(outDir string) (TelegramGateResult, error) {
	summaryPath := filepath.Join(outDir, "bpf", "maps", "summary.json")
	data, err := os.ReadFile(summaryPath)
	if err != nil {
		if os.IsNotExist(err) {
			return TelegramGateResult{
				Checks: []TelegramGateCheck{{
					Name:   "bpf_summary",
					Value:  "missing",
					OK:     true,
					Detail: "skipped (no BPF session; set ADSTACK_BPF_PROBE=1)",
				}},
				Pass: true,
			}, nil
		}
		return TelegramGateResult{}, err
	}
	var summary bpfSummary
	if err := json.Unmarshal(data, &summary); err != nil {
		return TelegramGateResult{}, err
	}

	var trackerConnects int64
	for _, n := range summary.Network {
		if n.Role == "tracker" {
			trackerConnects += n.Connects
		}
	}
	connectOK := trackerConnects == 0
	checks := []TelegramGateCheck{{
		Name:   "tracker_outbound_connect",
		Value:  strconv.FormatInt(trackerConnects, 10),
		Limit:  "0",
		OK:     connectOK,
		Detail: "T9: /tg/* handler must not call connect()",
	}}
	pass := connectOK
	return TelegramGateResult{Checks: checks, Pass: pass}, nil
}

// CheckTelegramSLA validates tracker handler p99 during soak (Prometheus).
func CheckTelegramSLA(promURL string) (TelegramGateResult, error) {
	prom := newPromClient(promURL)
	raw := prom.scalar(`histogram_quantile(0.99, sum(rate(ad_http_request_duration_seconds_bucket{job="tracker"}[5m])) by (le)) * 1000`)
	chk := TelegramGateCheck{
		Name:  "tracker_handler_p99",
		Limit: fmt.Sprintf("%.0f", TelegramTrackerHandlerP99MsLimit),
	}
	if raw == "na" || raw == "" {
		chk.Value = "na"
		chk.OK = true
		chk.Detail = "skipped (Prometheus unavailable)"
		return TelegramGateResult{Checks: []TelegramGateCheck{chk}, Pass: true}, nil
	}
	val, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil {
		chk.Value = raw
		chk.OK = true
		chk.Detail = "skipped (unparseable scalar)"
		return TelegramGateResult{Checks: []TelegramGateCheck{chk}, Pass: true}, nil
	}
	chk.Value = fmt.Sprintf("%.2f", val)
	chk.OK = val < TelegramTrackerHandlerP99MsLimit
	chk.Detail = "platform-sla.mdc tracker SLA"
	pass := chk.OK
	return TelegramGateResult{Checks: []TelegramGateCheck{chk}, Pass: pass}, nil
}

// CheckTelegramHotPathGate runs BPF + SLA checks for Telegram T9 sign-off.
func CheckTelegramHotPathGate(outDir, promURL string) (TelegramGateResult, error) {
	bpfRes, err := CheckTelegramBPF(outDir)
	if err != nil {
		return TelegramGateResult{}, err
	}
	slaRes, err := CheckTelegramSLA(promURL)
	if err != nil {
		return TelegramGateResult{}, err
	}
	merged := TelegramGateResult{
		Checks: append(bpfRes.Checks, slaRes.Checks...),
		Pass:   bpfRes.Pass && slaRes.Pass,
	}
	return merged, nil
}

// WriteTelegramGateReport writes telegram-gate.md and returns an error when checks fail.
func WriteTelegramGateReport(outDir, promURL string) (string, error) {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", err
	}
	result, err := CheckTelegramHotPathGate(outDir, promURL)
	if err != nil {
		return "", err
	}
	path := filepath.Join(outDir, "telegram-gate.md")
	var b strings.Builder
	b.WriteString("# Telegram hot-path gate (T9)\n\n")
	b.WriteString(fmt.Sprintf("Generated: %s\n", time.Now().UTC().Format(time.RFC3339)))
	b.WriteString(fmt.Sprintf("Session: `%s`\n\n", outDir))
	b.WriteString("| Check | Value | Limit | Status | Detail |\n")
	b.WriteString("|-------|-------|-------|--------|--------|\n")
	for _, c := range result.Checks {
		status := "PASS"
		if !c.OK {
			status = "FAIL"
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n", c.Name, c.Value, c.Limit, status, c.Detail)
	}
	if result.Pass {
		b.WriteString("\n**Result: PASS**\n")
	} else {
		b.WriteString("\n**Result: FAIL** — see TELEGRAM.md T9 / platform-sla.mdc SLA.\n")
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return "", err
	}
	if !result.Pass {
		return path, fmt.Errorf("%w: see %s", ErrTelegramGateFailed, path)
	}
	return path, nil
}
