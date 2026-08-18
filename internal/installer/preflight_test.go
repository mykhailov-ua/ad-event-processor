package installer

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bidshard/ad-event-processor/pkg/naming"
)

func TestProfileValidation(t *testing.T) {
	btfFile := writeTempFile(t, "vmlinux", "btf")

	oldBTF := btfPathOverride
	btfPathOverride = btfFile
	t.Cleanup(func() {
		btfPathOverride = oldBTF
	})

	tests := []struct {
		name    string
		profile InstallProfile
		wantErr bool
	}{
		{
			name: "valid single_vps",
			profile: InstallProfile{
				Type:      ProfileSingleVPS,
				Interface: "eth0",
			},
		},
		{
			name: "invalid profile",
			profile: InstallProfile{
				Type: "invalid",
			},
			wantErr: true,
		},
		{
			name: "edge_xdp in compose_dev",
			profile: InstallProfile{
				Type:    ProfileComposeDev,
				EdgeXDP: true,
			},
			wantErr: true,
		},
		{
			name: "multi_region valid single_vps",
			profile: InstallProfile{
				Type:        ProfileSingleVPS,
				Interface:   "eth0",
				MultiRegion: true,
			},
		},
		{
			name: "multi_region blocked in compose_dev",
			profile: InstallProfile{
				Type:        ProfileComposeDev,
				MultiRegion: true,
			},
			wantErr: true,
		},
		{
			name: "edge_xdp without btf",
			profile: InstallProfile{
				Type:      ProfileSingleVPS,
				Interface: "eth0",
				EdgeXDP:   true,
			},
			wantErr: true,
		},
		{
			name: "ingress_schema defaults to openrtb_3",
			profile: InstallProfile{
				Type:      ProfileSingleVPS,
				Interface: "eth0",
			},
		},
		{
			name: "ingress_schema legacy_native",
			profile: InstallProfile{
				Type:          ProfileComposeDev,
				IngressSchema: legacyIngressNativeSchema(),
			},
		},
		{
			name: "ingress_schema invalid",
			profile: InstallProfile{
				Type:          ProfileComposeDev,
				IngressSchema: "custom_json",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := tt.profile
			if tt.name == "edge_xdp without btf" {
				btfPathOverride = filepath.Join(t.TempDir(), "missing")
			} else {
				btfPathOverride = btfFile
			}

			err := profile.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && tt.name == "ingress_schema defaults to openrtb_3" {
				if profile.IngressSchema != IngressSchemaOpenRTB3 {
					t.Fatalf("expected default ingress_schema openrtb_3, got %s", profile.IngressSchema)
				}
			}
		})
	}
}

func TestPreflightBTF(t *testing.T) {
	btfFile := writeTempFile(t, "vmlinux", "btf")
	old := btfPathOverride
	btfPathOverride = btfFile
	t.Cleanup(func() { btfPathOverride = old })

	res := checkBTF()
	if res.Status != StatusPass {
		t.Fatalf("expected PASS, got %s (%s)", res.Status, res.Message)
	}
}

func TestPreflightNICWithFakeEthtool(t *testing.T) {
	old := ethToolOutputOverride
	ethToolOutputOverride = "driver: ixgbe\n"
	t.Cleanup(func() { ethToolOutputOverride = old })

	res := checkNIC()
	if res.Status != StatusPass {
		t.Fatalf("expected PASS, got %s (%s)", res.Status, res.Message)
	}
}

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

	if string(got) != string(want) {
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

	if string(first) != string(second) {
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

func writeTempFile(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}
