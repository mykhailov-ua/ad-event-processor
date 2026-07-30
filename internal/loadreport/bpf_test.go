package loadreport_test

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"espx/internal/loadreport"
)

func TestWriteBPFReport_diskGateFixture(t *testing.T) {
	fixture := filepath.Join(moduleRoot(t), "testdata", "bpf_disk_gate")
	out := t.TempDir()
	if err := copyDir(filepath.Join(fixture, "bpf"), filepath.Join(out, "bpf")); err != nil {
		t.Fatal(err)
	}

	reportPath, err := loadreport.WriteBPFReport(out)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{
		"Group-commit coalescing: PASS",
		"writev",
		"sync reduction vs 1:1 baseline: 98.4%",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("report missing %q\n%s", want, text)
		}
	}
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		return copyFile(path, target, info.Mode())
	})
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
