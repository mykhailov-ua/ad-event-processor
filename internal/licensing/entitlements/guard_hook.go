package entitlements

var guardTrippedFn func() bool

func SetGuardTrippedHook(fn func() bool) {
	guardTrippedFn = fn
}

func guardTripped() bool {
	if guardTrippedFn == nil {
		return false
	}
	return guardTrippedFn()
}
