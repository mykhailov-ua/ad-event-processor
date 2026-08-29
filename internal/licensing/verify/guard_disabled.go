//go:build !linux || !license_guard

package verify

import "context"

func GuardCompiledIn() bool { return false }

func MaybeRunGuardWatchdogCLI([]string) bool { return false }

func StartLicenseGuard(context.Context, GuardConfig) {}

func RunGuardProbeForTest() bool { return false }

func resetGuardHooksForTest() {}
