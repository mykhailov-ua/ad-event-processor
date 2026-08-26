package controlplane

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
)

type FlowService interface {
	CreateLander(ctx context.Context, req CreateLanderRequest) (LanderDTO, error)
	ListLanders(ctx context.Context) ([]LanderDTO, error)
	UploadHostedLanderZip(ctx context.Context, landerID uuid.UUID, zipReader io.ReaderAt, zipSize int64) (LanderDTO, error)
	ServeHostedLanderFile(ctx context.Context, landerID uuid.UUID, relPath string) (io.ReadCloser, string, error)
	CreateOffer(ctx context.Context, req CreateOfferRequest) (OfferDTO, error)
	ListOffers(ctx context.Context) ([]OfferDTO, error)
	CreateFlow(ctx context.Context, req CreateFlowRequest) (FlowDTO, error)
	ListFlows(ctx context.Context) ([]FlowDTO, error)
	GetFlow(ctx context.Context, flowID uuid.UUID) (FlowDTO, error)
	UpdateFlow(ctx context.Context, flowID uuid.UUID, req UpdateFlowRequest) (FlowDTO, error)
}

type LanderDTO struct {
	ID            uuid.UUID  `json:"id"`
	Name          string     `json:"name"`
	URL           string     `json:"url,omitempty"`
	HostedAssetID *uuid.UUID `json:"hosted_asset_id,omitempty"`
	HostedURL     string     `json:"hosted_url,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

type OfferDTO struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateLanderRequest struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type CreateOfferRequest struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type FlowPathLanderRef struct {
	LanderID uuid.UUID `json:"lander_id"`
	Weight   int32     `json:"weight"`
}

type FlowPathOfferRef struct {
	OfferID  uuid.UUID `json:"offer_id"`
	Weight   int32     `json:"weight"`
	CapDaily *int32    `json:"cap_daily,omitempty"`
	CapTotal *int32    `json:"cap_total,omitempty"`
}

type FlowPathDTO struct {
	Weight  int32               `json:"weight"`
	Landers []FlowPathLanderRef `json:"landers"`
	Offers  []FlowPathOfferRef  `json:"offers"`
	Filters *FlowPathFiltersDTO `json:"filters,omitempty"`
}

type FlowDTO struct {
	ID        uuid.UUID       `json:"id"`
	Name      string          `json:"name"`
	Paths     json.RawMessage `json:"paths"`
	CreatedAt time.Time       `json:"created_at"`
}

type CreateFlowRequest struct {
	Name  string        `json:"name"`
	Paths []FlowPathDTO `json:"paths"`
}

type FlowHTTPHandlers struct {
	Service           FlowService
	ApplyRateLimit    func(http.HandlerFunc) http.HandlerFunc
	RequirePermission func(string, http.HandlerFunc) http.HandlerFunc
}

func (h *FlowHTTPHandlers) Register(mux *http.ServeMux) {
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
	mux.HandleFunc("GET /api/v1/offers", limit(perm("campaigns:read", h.listOffers)))
	mux.HandleFunc("POST /api/v1/offers", limit(perm("campaigns:write", h.createOffer)))
	mux.HandleFunc("GET /api/v1/flows", limit(perm("campaigns:read", h.listFlows)))
	mux.HandleFunc("POST /api/v1/flows", limit(perm("campaigns:write", h.createFlow)))
	mux.HandleFunc("GET /api/v1/flows/{id}", limit(perm("campaigns:read", h.getFlow)))
	mux.HandleFunc("PUT /api/v1/flows/{id}", limit(perm("campaigns:write", h.updateFlow)))
	h.RegisterHostedLanderRoutes(mux)
}

func (h *FlowHTTPHandlers) listLanders(w http.ResponseWriter, r *http.Request) {
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

func (h *FlowHTTPHandlers) createLander(w http.ResponseWriter, r *http.Request) {
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

func (h *FlowHTTPHandlers) listOffers(w http.ResponseWriter, r *http.Request) {
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

func (h *FlowHTTPHandlers) createOffer(w http.ResponseWriter, r *http.Request) {
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

func (h *FlowHTTPHandlers) listFlows(w http.ResponseWriter, r *http.Request) {
	items, err := h.Service.ListFlows(r.Context())
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if items == nil {
		items = []FlowDTO{}
	}
	httpresponse.JSON(w, http.StatusOK, items)
}

func (h *FlowHTTPHandlers) createFlow(w http.ResponseWriter, r *http.Request) {
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

func (h *FlowHTTPHandlers) getFlow(w http.ResponseWriter, r *http.Request) {
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

func (h *FlowHTTPHandlers) updateFlow(w http.ResponseWriter, r *http.Request) {
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
