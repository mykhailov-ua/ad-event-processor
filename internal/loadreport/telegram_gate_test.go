package loadreport

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckTelegramBPF_trackerConnectZero(t *testing.T) {
	dir := filepath.Join("testdata", "telegram_gate_pass")
	res, err := CheckTelegramBPF(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Pass {
		t.Fatalf("expected pass, got %+v", res.Checks)
	}
}

func TestCheckTelegramBPF_ignoreUnixRedisInfraConnects(t *testing.T) {
	dir := t.TempDir()
	bpfDir := filepath.Join(dir, "bpf", "maps")
	if err := os.MkdirAll(bpfDir, 0o755); err != nil {
		t.Fatal(err)
	}
	summary := `{"duration_sec":1,"network":[{"name":"tracker","role":"tracker","dport":0,"connects":1371}]}`
	if err := os.WriteFile(filepath.Join(bpfDir, "summary.json"), []byte(summary), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := CheckTelegramBPF(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Pass {
		t.Fatalf("expected pass for unix infra connects, got %+v", res.Checks)
	}
}

func TestCheckTelegramBPF_failOnTrackerConnect(t *testing.T) {
	dir := t.TempDir()
	bpfDir := filepath.Join(dir, "bpf", "maps")
	if err := os.MkdirAll(bpfDir, 0o755); err != nil {
		t.Fatal(err)
	}
	summary := `{"duration_sec":1,"network":[{"name":"tracker","role":"tracker","dport":443,"connects":3}]}`
	if err := os.WriteFile(filepath.Join(bpfDir, "summary.json"), []byte(summary), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := CheckTelegramBPF(dir)
	if err != nil {
		t.Fatal(err)
	}
	if res.Pass {
		t.Fatal("expected fail when tracker has outbound connects")
	}
}
