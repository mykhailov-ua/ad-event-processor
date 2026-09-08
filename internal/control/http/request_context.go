package http

import (
	"net/http"
	"strings"
	"time"

	"ad-event-processor/pkg/coldpath"
)

// AdminRequestContextMiddleware bounds /api/* handler context for Postgres and outbound cancel propagation.
// longRoutes maps "METHOD /path" to a longer ceiling than defaultTimeout (cost-sync run, migration pull, support bundle).
func AdminRequestContextMiddleware(defaultTimeout time.Duration, longRoutes map[string]time.Duration) func(http.Handler) http.Handler {
	if defaultTimeout <= 0 {
		defaultTimeout = 10 * time.Second
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasPrefix(r.URL.Path, "/api/") {
				next.ServeHTTP(w, r)
				return
			}
			timeout := defaultTimeout
			if len(longRoutes) > 0 {
				if long, ok := longRoutes[r.Method+" "+r.URL.Path]; ok && long > 0 {
					timeout = long
				}
			}
			ctx, cancel := coldpath.BoundedContext(r.Context(), timeout)
			defer cancel()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
