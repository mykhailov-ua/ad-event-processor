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

func registerRootHandler(mux *http.ServeMux) {
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		httpresponse.JSON(w, http.StatusNotFound, map[string]any{
			"error": map[string]string{
				"code":    "NOT_FOUND",
				"message": "no UI at /; use JSON /api/v1 or bundle the admin SPA (see docs/SELF_HOSTED.md#ui-no-server-side-htmx)",
			},
		})
	})
}
