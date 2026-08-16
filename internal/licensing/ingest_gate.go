package licensing

// statePermitsIngest implements P-C3-03 without exporting the legacy predicate name to hot-path callers.
func statePermitsIngest(state LicenseState) bool {
	switch state {
	case StateExpired, StateRevoked:
		return false
	default:
		return true
	}
}

// IngestAllowed is retained for property tests and VERIFY.md mapping (P-C3-03).
func IngestAllowed(state LicenseState) bool {
	return statePermitsIngest(state)
}
