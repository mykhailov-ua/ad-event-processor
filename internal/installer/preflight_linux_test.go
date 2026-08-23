//go:build linux

package installer

import (
	"os"
	"path/filepath"
	"testing"
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

func writeTempFile(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}
