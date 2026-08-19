package loadreport

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCompareBPFSnapshots_regressionFails(t *testing.T) {
	baseline := BPFSnapshot{Metrics: map[string]float64{
		"filter_check_uprobe_p99_us": 300,
		"max_fd_delta":               0,
	}}
	current := BPFSnapshot{Metrics: map[string]float64{
		"filter_check_uprobe_p99_us": 400,
		"max_fd_delta":               0,
	}}
	res := CompareBPFSnapshots(baseline, current)
	if res.Pass {
		t.Fatal("expected fail on +33% filter_check p99")
	}
}

func TestCompareBPFSnapshots_fdDeltaRegression(t *testing.T) {
	baseline := BPFSnapshot{Metrics: map[string]float64{"max_fd_delta": 0}}
	current := BPFSnapshot{Metrics: map[string]float64{"max_fd_delta": 3}}
	res := CompareBPFSnapshots(baseline, current)
	if res.Pass {
		t.Fatal("expected fail on fd delta increase")
	}
}

func TestCompareBPFSnapshots_rssDeltaRegression(t *testing.T) {
	baseline := BPFSnapshot{Metrics: map[string]float64{"max_rss_delta_kb": 0}}
	current := BPFSnapshot{Metrics: map[string]float64{"max_rss_delta_kb": 6144}}
	res := CompareBPFSnapshots(baseline, current)
	if res.Pass {
		t.Fatal("expected fail on rss delta increase")
	}
}

func TestSeedBPFSnapshotBaseline(t *testing.T) {
	dir := filepath.Join("testdata", "bpf_gate_pass")
	tmp := t.TempDir()
	if err := SeedBPFSnapshotBaseline(tmp, dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(tmp, "summary_snapshot.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(tmp, "summary.json")); err != nil {
		t.Fatal(err)
	}
}

func TestWriteBPFGateCompareReport_seedsBaseline(t *testing.T) {
	session := filepath.Join("testdata", "bpf_gate_pass")
	baseline := t.TempDir()
	path, err := WriteBPFGateCompareReport(baseline, session, "http://127.0.0.1:1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}
