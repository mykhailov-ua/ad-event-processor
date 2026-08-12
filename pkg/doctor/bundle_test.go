package doctor

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/internal/licensing"
)

func TestWriteBundle(t *testing.T) {
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_REQUIRED", "1")
	dir := t.TempDir()
	out := filepath.Join(dir, "bundle.tar.gz")
	err := WriteBundle(context.Background(), BundleOptions{
		Out:     out,
		Deps: ProbeDeps{
			LicenseDiagnostics: func() (licensing.LicenseDiagnostics, bool) {
				return licensing.LicenseDiagnostics{
					State:            licensing.StateActive,
					DeploymentID:     "dep-bundle",
					DaysToExpiry:     14,
					FingerprintMatch: true,
				}, true
			},
		},
		Only:    []string{"kernel"},
		Timeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(out)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	found := map[string]bool{}
	var license BundleLicenseDTO
	var gotLicense bool
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		found[hdr.Name] = true
		if hdr.Name == "doctor/license.json" {
			raw, err := io.ReadAll(tr)
			if err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(raw, &license); err != nil {
				t.Fatal(err)
			}
			gotLicense = true
		}
	}
	for _, want := range []string{"doctor/report.json", "doctor/checklist.json", "doctor/license.json", "version.json", "config/sanitized.env", "README.txt"} {
		if !found[want] {
			t.Fatalf("missing %s in bundle, have %v", want, found)
		}
	}
	if !gotLicense {
		t.Fatal("doctor/license.json not parsed")
	}
	if license.DeploymentID != "dep-bundle" || license.DaysToExpiry != 14 || !license.FingerprintMatch {
		t.Fatalf("unexpected license.json: %+v", license)
	}
}
