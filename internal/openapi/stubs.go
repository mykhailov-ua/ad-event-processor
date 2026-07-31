package openapi

var StubRoutes = map[string]struct{}{}

func IsStub(method, path string) bool {
	_, ok := StubRoutes[method+" "+path]
	return ok
}
