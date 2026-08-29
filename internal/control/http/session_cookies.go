package http

import (
	"net/http"
)

func WriteSessionCookies(w http.ResponseWriter, r *http.Request, accessToken, refreshToken string) error {
	setCookie(w, r, "accessToken", accessToken, "/", 3600, true)
	setCookie(w, r, "refreshToken", refreshToken, "/api/v1/auth", 30*24*3600, true)
	return rotateCSRFToken(w, r)
}
