//go:build linux && license_guard

package licensing_test

import (
	"context"
	"testing"

	"ad-event-processor/internal/licensing"
	"github.com/stretchr/testify/require"
)

func TestGuard_TracerPid(t *testing.T) {
	licensing.ResetGuardForTest()
	t.Cleanup(licensing.ResetGuardForTest)
	restore := licensing.SetGuardTracerPidReaderForTest(func() (int, error) { return 4242, nil })
	t.Cleanup(restore)

	require.False(t, licensing.GuardTripped())
	require.True(t, licensing.RunGuardProbeForTest())
	require.True(t, licensing.GuardTripped())
	require.True(t, licensing.LicenseEpochInvalid())
}

func TestGuard_SuspiciousMap(t *testing.T) {
	licensing.ResetGuardForTest()
	t.Cleanup(licensing.ResetGuardForTest)
	restorePID := licensing.SetGuardTracerPidReaderForTest(func() (int, error) { return 0, nil })
	t.Cleanup(restorePID)
	restoreMaps := licensing.SetGuardMapsScannerForTest(func() bool { return true })
	t.Cleanup(restoreMaps)

	require.True(t, licensing.RunGuardProbeForTest())
	require.True(t, licensing.GuardTripped())
}

func TestGuard_PtraceWatchdogHandshakeBusy(t *testing.T) {
	licensing.ResetGuardForTest()
	t.Cleanup(licensing.ResetGuardForTest)

	require.False(t, licensing.GuardTripped())
	licensing.ProcessGuardWatchdogHandshakeForTest("busy\n", nil)
	require.True(t, licensing.GuardTripped())
	require.True(t, licensing.LicenseEpochInvalid())
}

func TestGuard_PtraceWatchdogHandshakeOK(t *testing.T) {
	licensing.ResetGuardForTest()
	t.Cleanup(licensing.ResetGuardForTest)

	licensing.ProcessGuardWatchdogHandshakeForTest("ok\n", nil)
	require.False(t, licensing.GuardTripped())
}

func TestGuard_DisabledSkipsPtraceLauncher(t *testing.T) {
	licensing.ResetGuardForTest()
	t.Cleanup(licensing.ResetGuardForTest)

	var launched int32
	restore := licensing.SetGuardPtraceWatchdogLauncherForTest(func(context.Context) {
		launched++
	})
	t.Cleanup(restore)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	licensing.StartLicenseGuard(ctx, licensing.GuardConfig{
		Enabled:        false,
		PtraceWatchdog: true,
	})
	require.Equal(t, int32(0), launched)
}

func TestGuard_TextTamper(t *testing.T) {
	licensing.ResetGuardForTest()
	t.Cleanup(licensing.ResetGuardForTest)
	restorePID := licensing.SetGuardTracerPidReaderForTest(func() (int, error) { return 0, nil })
	t.Cleanup(restorePID)
	restoreMaps := licensing.SetGuardMapsScannerForTest(func() bool { return false })
	t.Cleanup(restoreMaps)

	base := [32]byte{1, 2, 3, 4}
	tampered := [32]byte{9, 9, 9, 9}
	licensing.SetGuardTextBaselineForTest(base)
	restoreHash := licensing.SetGuardTextHasherForTest(func() ([32]byte, error) {
		return tampered, nil
	})
	t.Cleanup(restoreHash)

	require.True(t, licensing.RunGuardProbeForTest())
	require.True(t, licensing.GuardTripped())
}
