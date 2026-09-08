package flow

import (
	"net/http"
	"strings"

	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"
)

func (h *HTTPHandlers) getOffer(w http.ResponseWriter, r *http.Request) {
	id, err := coldpath.ParsePathUUID(r, "id")
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid offer id")
		return
	}
	dto, err := h.Service.GetOffer(r.Context(), id)
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

func (h *HTTPHandlers) updateOffer(w http.ResponseWriter, r *http.Request) {
	id, err := coldpath.ParsePathUUID(r, "id")
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid offer id")
		return
	}
	req, ok := coldpath.DecodeRequestOrBadRequest[UpdateOfferRequest](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	dto, err := h.Service.UpdateOffer(r.Context(), id, req)
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

func (h *HTTPHandlers) deleteOffer(w http.ResponseWriter, r *http.Request) {
	id, err := coldpath.ParsePathUUID(r, "id")
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid offer id")
		return
	}
	err = h.Service.DeleteOffer(r.Context(), id)
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
