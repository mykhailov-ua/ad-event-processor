package supportbundle

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestBundleRedactionGolden(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "bundle_sample.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(filepath.Join("testdata", "bundle_no_secrets.golden"))
	if err != nil {
		t.Fatal(err)
	}
	got := RedactLog([]string{strings.TrimSpace(string(raw))})
	if !bytes.Equal(got, want) {
		t.Fatalf("redact mismatch:\ngot:\n%s\nwant:\n%s", got, want)
	}
	forbidden := []*regexp.Regexp{
		regexp.MustCompile(`(?i)https?://`),
		regexp.MustCompile(`sk_live`),
		regexp.MustCompile(`203\.0\.113\.10`),
		regexp.MustCompile(`lk_secret`),
	}
	for _, re := range forbidden {
		if re.Match(got) {
			t.Fatalf("forbidden pattern %q in output: %s", re.String(), got)
		}
	}
}

func TestWriteBundle(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "app.log")
	_ = os.WriteFile(logPath, []byte(`{"msg":"ping","ip":"10.0.0.1"}`+"\n"), 0o644)

	var buf bytes.Buffer
	err := Write(context.Background(), &buf, Options{
		Meta:     Meta{DeploymentID: "dep-1", LicenseState: "ACTIVE"},
		LogDir:   dir,
		MaxBytes: DefaultMaxBytes,
	})
	if err != nil {
		t.Fatal(err)
	}

	gz, err := gzip.NewReader(&buf)
	if err != nil {
		t.Fatal(err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	found := map[string]bool{}
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		found[hdr.Name] = true
		if hdr.Name == "logs/redacted.log" {
			body, _ := io.ReadAll(tr)
			if strings.Contains(string(body), "10.0.0.1") {
				t.Fatalf("ip not redacted: %s", body)
			}
		}
	}
	for _, name := range []string{"version.json", "config/sanitized.env", "goroutine.pprof", "heap.pprof", "logs/redacted.log"} {
		if !found[name] {
			t.Fatalf("missing %s in bundle", name)
		}
	}
}
