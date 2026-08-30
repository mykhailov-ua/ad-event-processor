package httpingress

func HTTP1MatchForceSafeHeader(key []byte) bool {
	return http1MatchForceSafeHeader(key)
}

func HTTP1ForceSafeValue(val []byte) bool {
	return http1ForceSafeValue(val)
}
