package flow

import (
	"net/http"
	"strings"

	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"
)

func (h *HTTPHandlers) getLander(w http.ResponseWriter, r *http.Request) {
	id, err := coldpath.ParsePathUUID(r, "id")
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid lander id")
		return
	}
	dto, err := h.Service.GetLander(r.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error())
			return
		}
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, dto)
}

func (h *HTTPHandlers) updateLander(w http.ResponseWriter, r *http.Request) {
	id, err := coldpath.ParsePathUUID(r, "id")
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid lander id")
		return
	}
	req, ok := coldpath.DecodeRequestOrBadRequest[UpdateLanderRequest](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	dto, err := h.Service.UpdateLander(r.Context(), id, req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error())
			return
		}
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, dto)
}

func (h *HTTPHandlers) deleteLander(w http.ResponseWriter, r *http.Request) {
	id, err := coldpath.ParsePathUUID(r, "id")
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid lander id")
		return
	}
	err = h.Service.DeleteLander(r.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error())
			return
		}
		if strings.Contains(err.Error(), "referenced") {
			httpresponse.Error(w, http.StatusConflict, "CONFLICT", err.Error())
			return
		}
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
