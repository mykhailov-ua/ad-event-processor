package management

import (
	"net/http"

	"espx/pkg/httpresponse"
)

func registerAdminGoneRoutes(mux *http.ServeMux) {
	gone := func(w http.ResponseWriter, r *http.Request) {
		httpresponse.Error(w, http.StatusGone, "GONE",
			"legacy /admin HTMX routes removed; use /api/v1 JSON API (see docs/SELF_HOSTED.md)")
	}
	for _, method := range []string{"GET", "POST", "PUT", "DELETE", "PATCH"} {
		mux.HandleFunc(method+" /admin/{path...}", gone)
	}
}

