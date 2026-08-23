package installer

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/bidshard/ad-event-processor/pkg/naming"
)

func TestPreflightJSONSchema(t *testing.T) {
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	_, runErr := RunPreflight(false, true)
	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	if runErr != nil {
		t.Fatalf("RunPreflight: %v", runErr)
	}

	var payload map[string]any
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json: %v", err)
	}

	for _, key := range []string{"checks", "passed"} {
		if _, ok := payload[key]; !ok {
			t.Fatalf("missing top-level field %q", key)
		}
	}

	checks, ok := payload["checks"].([]any)
	if !ok || len(checks) == 0 {
		t.Fatal("checks must be a non-empty array")
	}

	first, ok := checks[0].(map[string]any)
	if !ok {
		t.Fatal("check entry must be an object")
	}
	for _, key := range []string{"id", "description", "status"} {
		if _, ok := first[key]; !ok {
			t.Fatalf("missing check field %q", key)
		}
	}
}

func TestGoldenRenderSystemd(t *testing.T) {
	got, err := renderSystemdUnit(&InstallProfile{
		Type:          ProfileSingleVPS,
		Interface:     "eth0",
		IngressSchema: IngressSchemaOpenRTB3,
	})
	if err != nil {
		t.Fatal(err)
	}

	want, err := os.ReadFile("testdata/ad-event-processor-tracker.service")
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(got, want) {
		t.Fatalf("systemd unit mismatch:\n%s", strings.TrimSpace(string(got)))
	}
}

func TestIdempotentApply(t *testing.T) {
	root := t.TempDir()
	t.Setenv(naming.LegacyVendorEnvKey("INSTALL_ROOT"), root)

	profile := &InstallProfile{
		Type:             ProfileComposeDev,
		IngressSchema:    IngressSchemaOpenRTB3,
		TelemetryEnabled: false,
	}

	if err := renderTemplates(profile, false); err != nil {
		t.Fatalf("first apply: %v", err)
	}

	secrets := secretsPath()
	info, err := os.Stat(secrets)
	if err != nil {
		t.Fatalf("secrets missing: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("secrets mode = %o, want 0600", info.Mode().Perm())
	}

	first, err := os.ReadFile(secrets)
	if err != nil {
		t.Fatal(err)
	}

	if err := renderTemplates(profile, false); err != nil {
		t.Fatalf("second apply: %v", err)
	}

	second, err := os.ReadFile(secrets)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(first, second) {
		t.Fatal("second apply changed secrets content")
	}
}

func TestIdempotentDryRunOutput(t *testing.T) {
	profile := &InstallProfile{Type: ProfileComposeDev}
	out1 := captureRenderDryRun(t, profile)
	out2 := captureRenderDryRun(t, profile)
	if out1 != out2 {
		t.Fatalf("dry-run output changed between runs")
	}
}

func TestProvisionReadsPackagesYAML(t *testing.T) {
	path := packagesYAMLPath()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("packages.yaml missing: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "libpcap-dev") {
		t.Fatal("expected libpcap-dev in packages.yaml")
	}
}

func captureRenderDryRun(t *testing.T, profile *InstallProfile) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	if err := renderTemplates(profile, true); err != nil {
		t.Fatal(err)
	}
	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}
