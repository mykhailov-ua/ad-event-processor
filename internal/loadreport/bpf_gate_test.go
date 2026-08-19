package loadreport

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckBPFResourceGate_passFixture(t *testing.T) {
	dir := filepath.Join("testdata", "bpf_gate_pass")
	res, err := CheckBPFResourceGate(dir, "http://127.0.0.1:1")
	if err != nil {
		t.Fatal(err)
	}
	if !res.Pass {
		t.Fatalf("expected pass, checks: %+v", res.Checks)
	}
}

func TestCheckBPFResourceGate_failFilterCheckP99(t *testing.T) {
	dir := t.TempDir()
	writeBPFGateSummary(t, dir, `{
  "duration_sec": 60,
  "markers": [{"role":"tracker","marker":"filter_check","p99_us":2500,"count":10}],
  "network": [{"role":"tracker","connects":0}]
}`)
	res, err := CheckBPFResourceGate(dir, "http://127.0.0.1:1")
	if err != nil {
		t.Fatal(err)
	}
	if res.Pass {
		t.Fatal("expected fail on filter_check p99")
	}
}

func TestCheckBPFResourceGate_failTrackerConnect(t *testing.T) {
	dir := t.TempDir()
	writeBPFGateSummary(t, dir, `{
  "duration_sec": 60,
  "network": [{"role":"tracker","connects":2}]
}`)
	res, err := CheckBPFResourceGate(dir, "http://127.0.0.1:1")
	if err != nil {
		t.Fatal(err)
	}
	if res.Pass {
		t.Fatal("expected fail on tracker connects")
	}
}

func TestCheckBPFResourceGate_failTrackerRSS(t *testing.T) {
	dir := t.TempDir()
	writeBPFGateSummary(t, dir, `{
  "duration_sec": 60,
  "network": [{"role":"tracker","connects":0}],
  "proc_samples": [{"role":"tracker","rss_delta":8192,"maj_flt":0}]
}`)
	res, err := CheckBPFResourceGate(dir, "http://127.0.0.1:1")
	if err != nil {
		t.Fatal(err)
	}
	if res.Pass {
		t.Fatal("expected fail on tracker rss delta")
	}
}

func TestCheckBPFResourceGate_failMajorFaults(t *testing.T) {
	dir := t.TempDir()
	writeBPFGateSummary(t, dir, `{
  "duration_sec": 60,
  "network": [{"role":"tracker","connects":0}],
  "proc_samples": [{"role":"tracker","rss_delta":1024,"maj_flt":3}]
}`)
	res, err := CheckBPFResourceGate(dir, "http://127.0.0.1:1")
	if err != nil {
		t.Fatal(err)
	}
	if res.Pass {
		t.Fatal("expected fail on major faults")
	}
}

func TestCheckBPFResourceGate_missingSummarySkipsWhenNotStrict(t *testing.T) {
	t.Setenv("BPF_GATE_STRICT", "0")
	dir := t.TempDir()
	res, err := CheckBPFResourceGate(dir, "http://127.0.0.1:1")
	if err != nil {
		t.Fatal(err)
	}
	if !res.Pass {
		t.Fatalf("expected skip pass, got %+v", res.Checks)
	}
}

func TestCheckBPFResourceGate_missingSummaryFailsWhenStrict(t *testing.T) {
	t.Setenv("BPF_GATE_STRICT", "1")
	dir := t.TempDir()
	_, err := CheckBPFResourceGate(dir, "http://127.0.0.1:1")
	if err == nil {
		t.Fatal("expected error when strict and summary missing")
	}
}

func TestCheckBPFResourceGate_strictFailsWithoutUprobes(t *testing.T) {
	t.Setenv("BPF_GATE_STRICT", "1")
	dir := t.TempDir()
	writeBPFGateSummary(t, dir, `{
  "duration_sec": 60,
  "network": [{"role":"tracker","connects":0}]
}`)
	res, err := CheckBPFResourceGate(dir, "http://127.0.0.1:1")
	if err != nil {
		t.Fatal(err)
	}
	if res.Pass {
		t.Fatalf("expected fail without uprobes in strict mode, checks: %+v", res.Checks)
	}
}

func TestCheckBPFResourceGate_strictFailsWithoutPrometheus(t *testing.T) {
	t.Setenv("BPF_GATE_STRICT", "1")
	dir := filepath.Join("testdata", "bpf_gate_pass")
	res, err := CheckBPFResourceGate(dir, "http://127.0.0.1:1")
	if err != nil {
		t.Fatal(err)
	}
	if res.Pass {
		t.Fatalf("expected fail without Prometheus in strict mode, checks: %+v", res.Checks)
	}
}

func TestWriteBPFGateReport_writesArtifact(t *testing.T) {
	dir := filepath.Join("testdata", "bpf_gate_pass")
	path, err := WriteBPFGateReport(dir, "http://127.0.0.1:1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}

func writeBPFGateSummary(t *testing.T, dir, body string) {
	t.Helper()
	mapsDir := filepath.Join(dir, "bpf", "maps")
	if err := os.MkdirAll(mapsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mapsDir, "summary.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
