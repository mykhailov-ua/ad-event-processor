package http

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"strings"

	"ad-event-processor/pkg/httpresponse"
)

func GenerateSecureToken(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func NewCSRFMiddleware(adminAPIKey string) func(http.Handler) http.Handler {
	adminKeyBytes := []byte(adminAPIKey)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
				if !strings.HasPrefix(r.URL.Path, "/api/v1/") && !strings.HasPrefix(r.URL.Path, "/admin/") {
					next.ServeHTTP(w, r)
					return
				}

				if r.URL.Path == "/api/v1/auth/login" || r.URL.Path == "/api/v1/auth/refresh" || r.URL.Path == "/api/v1/auth/logout" ||
					r.URL.Path == "/api/v1/settings/platform/bootstrap" ||
					r.URL.Path == "/api/v1/public/activate" || r.URL.Path == "/api/v1/public/invite/accept" {
					next.ServeHTTP(w, r)
					return
				}

				if len(adminKeyBytes) > 0 {
					if key := r.Header.Get("X-Admin-API-Key"); key != "" &&
						subtle.ConstantTimeCompare([]byte(key), adminKeyBytes) == 1 {
						next.ServeHTTP(w, r)
						return
					}
				}

				cookie, err := r.Cookie("csrfToken")
				if err != nil || cookie.Value == "" {
					httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "forbidden: missing csrf cookie")
					return
				}

				headerToken := r.Header.Get("X-CSRF-Token")
				if headerToken == "" {
					httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "forbidden: missing csrf header")
					return
				}

				if subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(headerToken)) != 1 {
					httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "forbidden: invalid csrf token")
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}
