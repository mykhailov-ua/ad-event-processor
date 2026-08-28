package opsadmin

import (
	"net/http"

	"ad-event-processor/pkg/httpresponse"
)

func (h *HTTPHandlers) GetStackHealthSnapshot(w http.ResponseWriter, r *http.Request) {
	h.getStackHealthSnapshot(w, r)
}

func (h *HTTPHandlers) getStackHealthSnapshot(w http.ResponseWriter, r *http.Request) {
	snap, err := h.OpsReader.GetStackHealthSnapshot(r.Context())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, snap)
}
