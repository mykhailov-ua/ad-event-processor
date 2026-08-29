//go:build linux && license_guard

package verify_test

import (
	"context"
	"testing"

	"ad-event-processor/internal/licensing/verify"

	"github.com/stretchr/testify/require"
)

func TestGuard_TracerPid(t *testing.T) {
	verify.ResetGuardForTest()
	t.Cleanup(verify.ResetGuardForTest)
	restore := verify.SetGuardTracerPidReaderForTest(func() (int, error) { return 4242, nil })
	t.Cleanup(restore)

	require.False(t, verify.GuardTripped())
	require.True(t, verify.RunGuardProbeForTest())
	require.True(t, verify.GuardTripped())
	require.True(t, verify.LicenseEpochInvalid())
}

func TestGuard_SuspiciousMap(t *testing.T) {
	verify.ResetGuardForTest()
	t.Cleanup(verify.ResetGuardForTest)
	restorePID := verify.SetGuardTracerPidReaderForTest(func() (int, error) { return 0, nil })
	t.Cleanup(restorePID)
	restoreMaps := verify.SetGuardMapsScannerForTest(func() bool { return true })
	t.Cleanup(restoreMaps)

	require.True(t, verify.RunGuardProbeForTest())
	require.True(t, verify.GuardTripped())
}

func TestGuard_PtraceWatchdogHandshakeSkipFailsWhenRequired(t *testing.T) {
	verify.ResetGuardForTest()
	t.Cleanup(verify.ResetGuardForTest)
	verify.SetGuardPtraceRequiredForTest(true)
	t.Cleanup(func() { verify.SetGuardPtraceRequiredForTest(false) })

	require.False(t, verify.GuardTripped())
	verify.ProcessGuardWatchdogHandshakeForTest("skip\n", nil)
	require.True(t, verify.GuardTripped())
	require.True(t, verify.LicenseEpochInvalid())
}

func TestGuard_PtraceWatchdogHandshakeSkipIgnoredWhenOptional(t *testing.T) {
	verify.ResetGuardForTest()
	t.Cleanup(verify.ResetGuardForTest)

	verify.ProcessGuardWatchdogHandshakeForTest("skip\n", nil)
	require.False(t, verify.GuardTripped())
}

func TestGuard_PtraceWatchdogHandshakeBusy(t *testing.T) {
	verify.ResetGuardForTest()
	t.Cleanup(verify.ResetGuardForTest)

	require.False(t, verify.GuardTripped())
	verify.ProcessGuardWatchdogHandshakeForTest("busy\n", nil)
	require.True(t, verify.GuardTripped())
	require.True(t, verify.LicenseEpochInvalid())
}

func TestGuard_PtraceWatchdogHandshakeOK(t *testing.T) {
	verify.ResetGuardForTest()
	t.Cleanup(verify.ResetGuardForTest)

	verify.ProcessGuardWatchdogHandshakeForTest("ok\n", nil)
	require.False(t, verify.GuardTripped())
}

func TestGuard_DisabledSkipsPtraceLauncher(t *testing.T) {
	verify.ResetGuardForTest()
	t.Cleanup(verify.ResetGuardForTest)

	var launched int32
	restore := verify.SetGuardPtraceWatchdogLauncherForTest(func(context.Context) {
		launched++
	})
	t.Cleanup(restore)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	verify.StartLicenseGuard(ctx, verify.GuardConfig{
		Enabled:        false,
		PtraceWatchdog: true,
	})
	require.Equal(t, int32(0), launched)
}

func TestGuard_TripWithoutVerifyCall(t *testing.T) {
	verify.ResetGuardForTest()
	t.Cleanup(verify.ResetGuardForTest)
	restorePID := verify.SetGuardTracerPidReaderForTest(func() (int, error) { return 99, nil })
	t.Cleanup(restorePID)
	restoreMaps := verify.SetGuardMapsScannerForTest(func() bool { return false })
	t.Cleanup(restoreMaps)

	require.True(t, verify.RunGuardProbeForTest())
	require.True(t, verify.GuardTripped())
	require.True(t, verify.LicenseEpochInvalid())
}

func TestGuard_TextTamper(t *testing.T) {
	verify.ResetGuardForTest()
	t.Cleanup(verify.ResetGuardForTest)
	restorePID := verify.SetGuardTracerPidReaderForTest(func() (int, error) { return 0, nil })
	t.Cleanup(restorePID)
	restoreMaps := verify.SetGuardMapsScannerForTest(func() bool { return false })
	t.Cleanup(restoreMaps)

	base := [32]byte{1, 2, 3, 4}
	tampered := [32]byte{9, 9, 9, 9}
	verify.SetGuardTextBaselineForTest(base)
	restoreHash := verify.SetGuardTextHasherForTest(func() ([32]byte, error) {
		return tampered, nil
	})
	t.Cleanup(restoreHash)

	require.True(t, verify.RunGuardProbeForTest())
	require.True(t, verify.GuardTripped())
}

func TestGuard_TamperStretchBeforeTrip(t *testing.T) {
	verify.ResetGuardForTest()
	t.Cleanup(verify.ResetGuardForTest)
	restorePID := verify.SetGuardTracerPidReaderForTest(func() (int, error) { return 0, nil })
	t.Cleanup(restorePID)
	restoreMaps := verify.SetGuardMapsScannerForTest(func() bool { return false })
	t.Cleanup(restoreMaps)

	base := [32]byte{1, 2, 3, 4}
	tampered := [32]byte{9, 9, 9, 9}
	verify.SetGuardTextBaselineForTest(base)
	restoreHash := verify.SetGuardTextHasherForTest(func() ([32]byte, error) {
		return tampered, nil
	})
	t.Cleanup(restoreHash)

	var stretchedReason string
	restoreStretch := verify.SetGuardTamperStretchHookForTest(func(reason string, _ [32]byte) {
		stretchedReason = reason
	})
	t.Cleanup(restoreStretch)

	require.True(t, verify.RunGuardProbeForTest())
	require.Equal(t, "text_tamper", stretchedReason)
	require.True(t, verify.GuardTripped())
}
