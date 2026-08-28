package licensingadmin

import (
	"context"
	"net/http"

	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"
	"ad-event-processor/pkg/legal"
)

type EulaStatusDTO struct {
	Version  string `json:"version"`
	Accepted bool   `json:"accepted"`
	Required bool   `json:"required"`
	Text     string `json:"text,omitempty"`
}

type AcceptEulaRequest struct {
	Version string `json:"version"`
}

type EulaService interface {
	GetEulaStatus(ctx context.Context) (legal.Acceptance, bool, error)
	AcceptEula(ctx context.Context, version, acceptedBy string) error
}

type EulaHTTPHandlers struct {
	Service           EulaService
	ApplyRateLimit    func(http.HandlerFunc) http.HandlerFunc
	RequirePermission func(string, http.HandlerFunc) http.HandlerFunc
	WriteServiceError func(http.ResponseWriter, error)
}

func (h *EulaHTTPHandlers) Register(mux *http.ServeMux) {
	if h == nil || h.Service == nil {
		return
	}
	limit := h.ApplyRateLimit
	perm := h.RequirePermission
	if limit == nil {
		limit = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	if perm == nil {
		perm = func(_ string, next http.HandlerFunc) http.HandlerFunc { return next }
	}
	mux.HandleFunc("GET /api/v1/eula", limit(h.getEula))
	mux.HandleFunc("POST /api/v1/eula/accept", limit(perm("settings:write", h.postEulaAccept)))
}

func (h *EulaHTTPHandlers) getEula(w http.ResponseWriter, r *http.Request) {
	_, accepted, err := h.Service.GetEulaStatus(r.Context())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, EulaStatusDTO{
		Version:  legal.Version,
		Accepted: accepted,
		Required: !accepted,
		Text:     legal.Text,
	})
}

func (h *EulaHTTPHandlers) postEulaAccept(w http.ResponseWriter, r *http.Request) {
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}
	req, err := coldpath.DecodeBody[AcceptEulaRequest](body)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if err := h.Service.AcceptEula(r.Context(), req.Version, ""); err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, EulaStatusDTO{
		Version:  legal.Version,
		Accepted: true,
		Required: false,
	})
}

func (h *EulaHTTPHandlers) writeServiceError(w http.ResponseWriter, err error) {
	if h.WriteServiceError != nil {
		h.WriteServiceError(w, err)
		return
	}
	httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
}
