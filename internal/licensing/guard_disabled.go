//go:build !linux || !license_guard

package licensing

import "context"

func GuardCompiledIn() bool { return false }

func MaybeRunGuardWatchdogCLI([]string) bool { return false }

func StartLicenseGuard(context.Context, GuardConfig) {}

func RunGuardProbeForTest() bool { return false }

func resetGuardHooksForTest() {}

//nolint:unused // stub when license_guard is not compiled in
func recordGuardTrip(string) {}
