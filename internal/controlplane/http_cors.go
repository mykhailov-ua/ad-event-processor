package controlplane

import (
	"net/http"
	"strings"
)

// NewCORSMiddleware returns a CORS middleware.
// FIX [1.1]: wildcard "*" is no longer combined with Access-Control-Allow-Credentials:true.
// When the effective origin is "*" we reflect the request origin but omit the credentials header,
// preventing the RFC 6454-forbidden combination that some non-browser clients exploit.
func NewCORSMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	originsMap := make(map[string]bool)
	hasWildcard := false
	for _, o := range allowedOrigins {
		trimmed := strings.TrimSpace(o)
		if trimmed == "*" {
			hasWildcard = true
		}
		originsMap[trimmed] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			w.Header().Add("Vary", "Origin")

			if origin != "" {
				explicitMatch := originsMap[origin]
				wildcardMatch := hasWildcard && !explicitMatch

				if explicitMatch {
					// Explicit allowlist: credentials are safe.
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Access-Control-Allow-Credentials", "true")
					w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
					w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-CSRF-Token, X-Admin-API-Key")
				} else if wildcardMatch {
					// Wildcard: reflect origin but do NOT set Allow-Credentials.
					// This satisfies open CORS (e.g. public read APIs) without the
					// credentials leak: browsers will refuse cookies/auth on this path.
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
					w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-CSRF-Token, X-Admin-API-Key")
				}
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
