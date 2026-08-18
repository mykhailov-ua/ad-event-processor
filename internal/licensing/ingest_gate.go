package licensing

func statePermitsIngest(state LicenseState) bool {
	switch state {
	case StateExpired, StateRevoked:
		return false
	default:
		return true
	}
}

func IngestAllowed(state LicenseState) bool {
	return statePermitsIngest(state)
}
