package checkout

import (
	"net/http"

	"ad-event-processor/pkg/httpresponse"
)

func RegisterLegacyUIRoutes(mux *http.ServeMux) {
	gone := func(w http.ResponseWriter, r *http.Request) {
		httpresponse.Error(w, http.StatusGone, "GONE",
			"legacy payment HTML/HTMX UI removed; use payment gRPC or self-serve /api/v1 JSON API")
	}
	for _, method := range []string{"GET", "POST"} {
		mux.HandleFunc(method+" /ui/payment/{path...}", gone)
	}
}
