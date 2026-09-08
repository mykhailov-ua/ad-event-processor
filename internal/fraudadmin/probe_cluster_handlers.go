package fraudadmin

import (
	"errors"
	"net/http"
	"strings"

	"ad-event-processor/pkg/httpresponse"
)

func (h *HTTPHandlers) registerProbeClusterRoutes(mux *http.ServeMux, limit func(http.HandlerFunc) http.HandlerFunc, perm func(string, http.HandlerFunc) http.HandlerFunc) {
	if h == nil || h.ProbeClusters == nil {
		return
	}
	mux.HandleFunc("GET /api/v1/fraud/probe-clusters/{cluster_id}", limit(perm("audit:read", h.getProbeClusterSummary)))
}

func (h *HTTPHandlers) getProbeClusterSummary(w http.ResponseWriter, r *http.Request) {
	clusterID := strings.TrimSpace(r.PathValue("cluster_id"))
	dto, err := h.ProbeClusters.GetSummary(r.Context(), clusterID)
	if err != nil {
		if errors.Is(err, ErrProbeClusterNotFound) {
			httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", "probe cluster not found")
			return
		}
		if errors.Is(err, ErrValidation) {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid cluster_id")
			return
		}
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, dto)
}
