//go:build linux

package edge

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"ad-event-processor/internal/licensing"
	"ad-event-processor/pkg/faultproof"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestEdgeSealed_ValidLicenseLoadsCollection(t *testing.T) {
	requireBPF(t)

	cleanup := setupSealedLicenseFixture(t)
	defer cleanup()

	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_MODE", "enterprise")

	var objs EdgeObjects
	require.NoError(t, LoadEdgeObjectsLenient(&objs, nil))
	defer objs.Close()
	require.NotNil(t, objs.XdpEdgeFilter)
	require.NotNil(t, objs.BlocklistV4)
}

func TestEdgeSealed_XDPAttachMatchesBaseline(t *testing.T) {
	if os.Getenv("SEALED_BPF_XDP_SMOKE") != "1" {
		t.Skip("set SEALED_BPF_XDP_SMOKE=1 for sealed BPF XDP attach lab smoke")
	}
	requireBPF(t)
	if os.Geteuid() != 0 {
		t.Skip("root required for XDP attach on lo")
	}
	if _, err := os.Stat("/sys/kernel/btf/vmlinux"); err != nil {
		t.Skip("BTF vmlinux required for sealed BPF attach smoke")
	}

	cleanup := setupSealedLicenseFixture(t)
	defer cleanup()

	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_MODE", "dev")
	var baseline EdgeObjects
	require.NoError(t, LoadEdgeObjectsLenient(&baseline, nil))
	baselineAction := probeBlocklistDrop(t, &baseline)
	baselineLink := attachLoGeneric(t, baseline.XdpEdgeFilter)
	require.NoError(t, baselineLink.Close())
	baseline.Close()

	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_MODE", "enterprise")
	var sealed EdgeObjects
	require.NoError(t, LoadEdgeObjectsLenient(&sealed, nil))
	sealedAction := probeBlocklistDrop(t, &sealed)
	require.Equal(t, baselineAction, sealedAction, "sealed prog.Test must match unsealed baseline")

	sealedLink := attachLoGeneric(t, sealed.XdpEdgeFilter)
	info, err := sealedLink.Info()
	require.NoError(t, err)
	require.NotZero(t, info.ID, "sealed program attached on lo")
	require.NoError(t, sealedLink.Close())
	sealed.Close()

	faultproof.Log(t, "sealed_bpf_xdp_smoke", map[string]string{
		"harness":          "kernel_xdp_attach_lo_generic",
		"drop_assertion":   "prog_test_same_maps",
		"baseline_action":  fmt.Sprintf("%d", baselineAction),
		"sealed_action":    fmt.Sprintf("%d", sealedAction),
		"license_mode_dev": "unsealed_baseline",
		"license_mode_ent": "sealed_mck",
		"status":           "passed",
	})
}

func setupSealedLicenseFixture(t *testing.T) (cleanup func()) {
	t.Helper()

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	tel := licensing.HWIDTelemetry{
		DMIUUID:  "sealed-bpf-smoke-dmi",
		DiskID:   "sealed-bpf-smoke-disk",
		MAC:      "aa:bb:cc:dd:ee:01",
		CPUModel: "SealedBPF Smoke CPU",
		CPUCores: 8,
	}
	restoreHWID := licensing.SetHWIDCollectForTest(func() licensing.HWIDTelemetry { return tel })

	claims := licensing.LicenseClaims{
		Issuer:       "ad-event-processor-license",
		Subject:      uuid.NewString(),
		DeploymentID: uuid.NewString(),
		ValidFrom:    time.Now().Add(-time.Hour),
		ValidUntil:   time.Now().Add(24 * time.Hour),
	}
	claims.Bind.Mode = "hard"
	claims.HWIDHash = licensing.HashHWIDFromTelemetry(tel)

	token, err := licensing.SignJWT(claims, priv, licensing.DefaultLicenseKeyID)
	require.NoError(t, err)

	plain, err := edgePlaintextELF()
	if err != nil {
		restoreHWID()
		t.Skipf("edge bpf elf unavailable: %v (run go generate ./internal/edge/)", err)
	}

	mck, err := licensing.DeriveMCK(token, licensing.HostHWID())
	require.NoError(t, err)
	sealed, err := licensing.SealAsset(sealedEdgeAssetLabel, plain, mck)
	require.NoError(t, err)

	dir := t.TempDir()
	licensePath := filepath.Join(dir, "license.jwt")
	blobPath := filepath.Join(dir, "edge_sealed.bin")
	require.NoError(t, os.WriteFile(licensePath, []byte(token), 0o600))
	require.NoError(t, os.WriteFile(blobPath, sealed, 0o600))

	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_PUBLIC_KEY", hex.EncodeToString(pub))
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_PATH", licensePath)
	t.Setenv("AD_EVENT_PROCESSOR_EDGE_SEALED_BLOB", blobPath)

	return restoreHWID
}

func probeBlocklistDrop(t *testing.T, objs *EdgeObjects) uint32 {
	t.Helper()
	require.NoError(t, InitConfigWith(objs.Config, InitOptions{}))

	victim := net.IPv4(203, 0, 113, 77)
	require.NoError(t, objs.BlocklistHostV4.Update(HostKey(victim[0], victim[1], victim[2], victim[3]).Addr, uint8(1), ebpf.UpdateAny))

	pkt := buildSYNPacket(t, victim, net.IPv4(10, 0, 0, 1), trackerPort)
	ret, _, err := objs.XdpEdgeFilter.Test(pkt)
	require.NoError(t, err)
	return ret
}

func attachLoGeneric(t *testing.T, prog *ebpf.Program) link.Link {
	t.Helper()
	iface, err := net.InterfaceByName("lo")
	require.NoError(t, err)
	lnk, err := link.AttachXDP(link.XDPOptions{
		Program:   prog,
		Interface: iface.Index,
		Flags:     link.XDPGenericMode,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = lnk.Close() })
	return lnk
}
