package flow

import (
	"context"
	"io"
	"net/http"
	"strings"

	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
)

type Service interface {
	CreateLander(ctx context.Context, req CreateLanderRequest) (LanderDTO, error)
	ListLanders(ctx context.Context) ([]LanderDTO, error)
	GetLander(ctx context.Context, landerID uuid.UUID) (LanderDTO, error)
	UpdateLander(ctx context.Context, landerID uuid.UUID, req UpdateLanderRequest) (LanderDTO, error)
	DeleteLander(ctx context.Context, landerID uuid.UUID) error
	UploadHostedLanderZip(ctx context.Context, landerID uuid.UUID, zipReader io.ReaderAt, zipSize int64) (LanderDTO, error)
	ServeHostedLanderFile(ctx context.Context, landerID uuid.UUID, relPath string) (io.ReadCloser, string, error)
	CreateOffer(ctx context.Context, req CreateOfferRequest) (OfferDTO, error)
	ListOffers(ctx context.Context) ([]OfferDTO, error)
	GetOffer(ctx context.Context, offerID uuid.UUID) (OfferDTO, error)
	UpdateOffer(ctx context.Context, offerID uuid.UUID, req UpdateOfferRequest) (OfferDTO, error)
	DeleteOffer(ctx context.Context, offerID uuid.UUID) error
	CreateFlow(ctx context.Context, req CreateFlowRequest) (DTO, error)
	ListFlows(ctx context.Context) ([]DTO, error)
	GetFlow(ctx context.Context, flowID uuid.UUID) (DTO, error)
	UpdateFlow(ctx context.Context, flowID uuid.UUID, req UpdateFlowRequest) (DTO, error)
	DeleteFlow(ctx context.Context, flowID uuid.UUID) error
	CloneFlow(ctx context.Context, flowID uuid.UUID, req CloneFlowRequest) (DTO, error)
	InspectFlowPaths(ctx context.Context, paths []PathDTO) (ValidateResponseDTO, error)
}

type HTTPHandlers struct {
	Service           Service
	ApplyRateLimit    func(http.HandlerFunc) http.HandlerFunc
	RequirePermission func(string, http.HandlerFunc) http.HandlerFunc
}

func (h *HTTPHandlers) Register(mux *http.ServeMux) {
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
	mux.HandleFunc("GET /api/v1/landers", limit(perm("campaigns:read", h.listLanders)))
	mux.HandleFunc("POST /api/v1/landers", limit(perm("campaigns:write", h.createLander)))
	mux.HandleFunc("GET /api/v1/landers/{id}", limit(perm("campaigns:read", h.getLander)))
	mux.HandleFunc("PATCH /api/v1/landers/{id}", limit(perm("campaigns:write", h.updateLander)))
	mux.HandleFunc("DELETE /api/v1/landers/{id}", limit(perm("campaigns:write", h.deleteLander)))
	mux.HandleFunc("GET /api/v1/offers", limit(perm("campaigns:read", h.listOffers)))
	mux.HandleFunc("POST /api/v1/offers", limit(perm("campaigns:write", h.createOffer)))
	mux.HandleFunc("GET /api/v1/offers/{id}", limit(perm("campaigns:read", h.getOffer)))
	mux.HandleFunc("PATCH /api/v1/offers/{id}", limit(perm("campaigns:write", h.updateOffer)))
	mux.HandleFunc("DELETE /api/v1/offers/{id}", limit(perm("campaigns:write", h.deleteOffer)))
	mux.HandleFunc("GET /api/v1/flows", limit(perm("campaigns:read", h.listFlows)))
	mux.HandleFunc("POST /api/v1/flows", limit(perm("campaigns:write", h.createFlow)))
	mux.HandleFunc("GET /api/v1/flows/{id}", limit(perm("campaigns:read", h.getFlow)))
	mux.HandleFunc("PUT /api/v1/flows/{id}", limit(perm("campaigns:write", h.updateFlow)))
	mux.HandleFunc("DELETE /api/v1/flows/{id}", limit(perm("campaigns:write", h.deleteFlow)))
	mux.HandleFunc("POST /api/v1/flows/{id}/clone", limit(perm("campaigns:write", h.cloneFlow)))
	mux.HandleFunc("POST /api/v1/flows/validate", limit(perm("campaigns:read", h.validateFlowPaths)))
	h.RegisterHostedLanderRoutes(mux)
}

func (h *HTTPHandlers) listLanders(w http.ResponseWriter, r *http.Request) {
	items, err := h.Service.ListLanders(r.Context())
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if items == nil {
		items = []LanderDTO{}
	}
	httpresponse.JSON(w, http.StatusOK, items)
}

func (h *HTTPHandlers) createLander(w http.ResponseWriter, r *http.Request) {
	req, ok := coldpath.DecodeRequestOrBadRequest[CreateLanderRequest](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	dto, err := h.Service.CreateLander(r.Context(), req)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusCreated, dto)
}

func (h *HTTPHandlers) listOffers(w http.ResponseWriter, r *http.Request) {
	items, err := h.Service.ListOffers(r.Context())
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if items == nil {
		items = []OfferDTO{}
	}
	httpresponse.JSON(w, http.StatusOK, items)
}

func (h *HTTPHandlers) createOffer(w http.ResponseWriter, r *http.Request) {
	req, ok := coldpath.DecodeRequestOrBadRequest[CreateOfferRequest](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	dto, err := h.Service.CreateOffer(r.Context(), req)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusCreated, dto)
}

func (h *HTTPHandlers) listFlows(w http.ResponseWriter, r *http.Request) {
	items, err := h.Service.ListFlows(r.Context())
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if items == nil {
		items = []DTO{}
	}
	httpresponse.JSON(w, http.StatusOK, items)
}

func (h *HTTPHandlers) createFlow(w http.ResponseWriter, r *http.Request) {
	req, ok := coldpath.DecodeRequestOrBadRequest[CreateFlowRequest](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	dto, err := h.Service.CreateFlow(r.Context(), req)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusCreated, dto)
}

func (h *HTTPHandlers) validateFlowPaths(w http.ResponseWriter, r *http.Request) {
	req, ok := coldpath.DecodeRequestOrBadRequest[struct {
		Paths []PathDTO `json:"paths"`
	}](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	resp, err := h.Service.InspectFlowPaths(r.Context(), req.Paths)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	status := http.StatusOK
	if !resp.Valid {
		status = http.StatusBadRequest
	}
	httpresponse.JSON(w, status, resp)
}

func (h *HTTPHandlers) cloneFlow(w http.ResponseWriter, r *http.Request) {
	id, err := coldpath.ParsePathUUID(r, "id")
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid flow id")
		return
	}
	req, ok := coldpath.DecodeRequestOrBadRequest[CloneFlowRequest](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	dto, err := h.Service.CloneFlow(r.Context(), id, req)
	if err != nil {
		if err.Error() == "flow not found" {
			httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error())
			return
		}
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusCreated, dto)
}

func (h *HTTPHandlers) getFlow(w http.ResponseWriter, r *http.Request) {
	id, err := coldpath.ParsePathUUID(r, "id")
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid flow id")
		return
	}
	dto, err := h.Service.GetFlow(r.Context(), id)
	if err != nil {
		if err.Error() == "flow not found" {
			httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error())
			return
		}
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, dto)
}

func (h *HTTPHandlers) updateFlow(w http.ResponseWriter, r *http.Request) {
	id, err := coldpath.ParsePathUUID(r, "id")
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid flow id")
		return
	}
	req, ok := coldpath.DecodeRequestOrBadRequest[UpdateFlowRequest](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	dto, err := h.Service.UpdateFlow(r.Context(), id, req)
	if err != nil {
		if err.Error() == "flow not found" {
			httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error())
			return
		}
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, dto)
}

func (h *HTTPHandlers) deleteFlow(w http.ResponseWriter, r *http.Request) {
	id, err := coldpath.ParsePathUUID(r, "id")
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid flow id")
		return
	}
	err = h.Service.DeleteFlow(r.Context(), id)
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
