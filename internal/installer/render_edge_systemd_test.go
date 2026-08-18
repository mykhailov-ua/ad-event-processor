package installer

import (
	"os"
	"strings"
	"testing"
)

func TestGoldenRenderEdgeSystemdUnits(t *testing.T) {
	profile := &InstallProfile{
		Type:      ProfileSingleVPS,
		Interface: "eth0",
		EdgeXDP:   true,
	}

	xdp, err := renderEdgeXDPUnit(profile)
	if err != nil {
		t.Fatal(err)
	}
	wantXDP, err := os.ReadFile("testdata/ad-event-processor-edge-xdp.service")
	if err != nil {
		t.Fatal(err)
	}
	if string(xdp) != string(wantXDP) {
		t.Fatalf("edge-xdp unit mismatch:\n%s", strings.TrimSpace(string(xdp)))
	}

	sync, err := renderEdgeBPFSyncUnit(profile)
	if err != nil {
		t.Fatal(err)
	}
	wantSync, err := os.ReadFile("testdata/ad-event-processor-edge-bpf-sync.service")
	if err != nil {
		t.Fatal(err)
	}
	if string(sync) != string(wantSync) {
		t.Fatalf("edge-bpf-sync unit mismatch:\n%s", strings.TrimSpace(string(sync)))
	}
}

func TestEdgeSystemdManifests_skippedWhenDisabled(t *testing.T) {
	manifests, err := edgeSystemdManifests(&InstallProfile{
		Type:      ProfileSingleVPS,
		Interface: "eth0",
		EdgeXDP:   false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(manifests) != 0 {
		t.Fatalf("expected no edge manifests, got %d", len(manifests))
	}
}

func TestSyncEdgeSystemdUnits_dryRun(t *testing.T) {
	if err := syncEdgeSystemdUnits(&InstallProfile{
		Type:      ProfileSingleVPS,
		Interface: "eth0",
		EdgeXDP:   true,
	}, true); err != nil {
		t.Fatal(err)
	}
}
