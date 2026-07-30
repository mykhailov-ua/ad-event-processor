package management

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"espx/pkg/supportbundle"
)

func TestBundleRedaction(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "pkg", "supportbundle", "testdata", "bundle_sample.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(filepath.Join("..", "..", "pkg", "supportbundle", "testdata", "bundle_no_secrets.golden"))
	if err != nil {
		t.Fatal(err)
	}
	got := supportbundle.RedactLog([]string{strings.TrimSpace(string(raw))})
	if string(got) != string(want) {
		t.Fatalf("redact mismatch:\n--- got ---\n%s--- want ---\n%s", got, want)
	}
	forbidden := []*regexp.Regexp{
		regexp.MustCompile(`(?i)https?://`),
		regexp.MustCompile(`sk_live`),
		regexp.MustCompile(`203\.0\.113\.10`),
		regexp.MustCompile(`lk_secret`),
		regexp.MustCompile(`(?i)pii_salt_hex=deadbeef`),
	}
	for _, re := range forbidden {
		if re.Match(got) {
			t.Fatalf("forbidden pattern %q in output: %s", re.String(), got)
		}
	}
}
