package controlplane

import (
	"net/http"
	"strings"
)

func contentSecurityPolicy(path string) string {
	frame := "frame-ancestors 'none'"
	if strings.HasPrefix(path, "/api/") {
		return "default-src 'none'; " + frame
	}
	if path == "/login" || path == "/bootstrap" || strings.HasPrefix(path, "/assets/") || isAdminSPAPath(path) {
		return "default-src 'none'; script-src 'self'; style-src 'self'; connect-src 'self'; img-src 'self' data:; base-uri 'self'; form-action 'self'; " + frame
	}
	return "default-src 'none'; " + frame
}

func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Content-Security-Policy", contentSecurityPolicy(r.URL.Path))
		if r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		next.ServeHTTP(w, r)
	})
}
