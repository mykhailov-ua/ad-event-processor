package adminapi

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/bidshard/ad-event-processor/internal/openrtb"
	"github.com/bidshard/ad-event-processor/pkg/coldpath"
	"github.com/bidshard/ad-event-processor/pkg/httpresponse"
	"github.com/bidshard/ad-event-processor/pkg/platformconfig"
)

type RtbDealDTO struct {
	ID         int64  `json:"id"`
	DealID     string `json:"deal_id"`
	FloorMicro int64  `json:"floor_micro"`
	GeoMask    int64  `json:"geo_mask"`
	CatMask    int64  `json:"cat_mask"`
	Pacing     string `json:"pacing"`
	Seats      int32  `json:"seats"`
	CustomerID string `json:"customer_id"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

type RtbDealCreateSpec struct {
	DealID     string `json:"deal_id"`
	FloorMicro int64  `json:"floor_micro"`
	GeoMask    int64  `json:"geo_mask"`
	CatMask    int64  `json:"cat_mask"`
	Pacing     string `json:"pacing"`
	Seats      int32  `json:"seats"`
	CustomerID string `json:"customer_id"`
}

type RtbDealUpdateSpec struct {
	DealID     string `json:"deal_id"`
	FloorMicro int64  `json:"floor_micro"`
	GeoMask    int64  `json:"geo_mask"`
	CatMask    int64  `json:"cat_mask"`
	Pacing     string `json:"pacing"`
	Seats      int32  `json:"seats"`
	CustomerID string `json:"customer_id"`
}

type RtbService interface {
	ListRtbDeals(ctx context.Context) ([]RtbDealDTO, error)
	GetRtbDeal(ctx context.Context, id int64) (RtbDealDTO, error)
	CreateRtbDeal(ctx context.Context, spec RtbDealCreateSpec) (RtbDealDTO, error)
	UpdateRtbDeal(ctx context.Context, id int64, spec RtbDealUpdateSpec) (RtbDealDTO, error)
	DeleteRtbDeal(ctx context.Context, id int64) error
}

type RtbShadowDiffSnapshotDTO struct {
	Window            string  `json:"window"`
	Source            string  `json:"source"`
	ShadowEvals       uint64  `json:"shadow_evals"`
	ShadowWinnerMatch uint64  `json:"shadow_winner_match"`
	ShadowMismatch    uint64  `json:"shadow_winner_mismatch"`
	ShadowNoBid       uint64  `json:"shadow_no_bid"`
	LiveWouldAccept   uint64  `json:"live_would_accept"`
	LiveWouldReject   uint64  `json:"live_would_reject"`
	ParityMatch       uint64  `json:"parity_match"`
	ParityRate        float64 `json:"parity_rate"`
	MismatchRate      float64 `json:"mismatch_rate"`
}

type RtbLiveGateDTO struct {
	Ready   bool                     `json:"ready"`
	Reasons []string                 `json:"reasons,omitempty"`
	Shadow  RtbShadowDiffSnapshotDTO `json:"shadow"`
}

type RtbHTTPHandlers struct {
	Service           RtbService
	ApplyRateLimit    func(http.HandlerFunc) http.HandlerFunc
	RequirePermission func(string, http.HandlerFunc) http.HandlerFunc
	WriteServiceError func(http.ResponseWriter, error)
	ExchangeConfig    openrtb.ExchangeConfig
	ShadowDiff        func(time.Duration) RtbShadowDiffSnapshotDTO
	LiveGate          func(time.Duration) RtbLiveGateDTO
	ReconcileCH       RtbReconcileCHFunc
	RuntimeConfig     RtbRuntimeConfigReader
	PlatformConfig    PlatformConfigReader
}

// RtbRuntimeConfigReader supplies env-driven tracker/exchange hints for the admin UI.
type RtbRuntimeConfigReader interface {
	RtbMode() string
	RtbEnabled() bool
	RtbExchangeNoBidMode() string
}

type RtbRuntimeHintsDTO struct {
	RtbMode              string `json:"rtb_mode"`
	RtbEnabled           bool   `json:"rtb_enabled"`
	RtbExchangeNoBidMode string `json:"rtb_exchange_no_bid_mode,omitempty"`
}

type RtbEndpointsDTO struct {
	OpenRTBBidURL     string `json:"openrtb_bid_url"`
	EdgeExposeOpenRTB bool   `json:"edge_expose_openrtb"`
	TrackingDomain    string `json:"tracking_domain,omitempty"`
	EdgePortHint      string `json:"edge_port_hint,omitempty"`
	TrackerPortHint   string `json:"tracker_port_hint,omitempty"`
}

type RtbIntegrationResponseDTO struct {
	openrtb.IntegrationProfile
	Runtime   RtbRuntimeHintsDTO `json:"runtime"`
	Endpoints RtbEndpointsDTO    `json:"endpoints"`
}

func (h *RtbHTTPHandlers) Register(mux *http.ServeMux) {
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
	mux.HandleFunc("POST /api/v1/rtb/validate-bid-request", limit(perm("rtb:read", h.validateBidRequest)))
	mux.HandleFunc("GET /api/v1/rtb/integration-profile", limit(perm("rtb:read", h.integrationProfile)))
	mux.HandleFunc("GET /api/v1/rtb/shadow-diff", limit(perm("rtb:read", h.shadowDiff)))
	mux.HandleFunc("GET /api/v1/rtb/reconcile/export", limit(perm("rtb:read", h.reconcileExport)))
	mux.HandleFunc("GET /api/v1/rtb/deals", limit(perm("rtb:read", h.listDeals)))
	mux.HandleFunc("GET /api/v1/rtb/deals/{id}", limit(perm("rtb:read", h.getDeal)))
	mux.HandleFunc("POST /api/v1/rtb/deals", limit(perm("rtb:write", h.createDeal)))
	mux.HandleFunc("PATCH /api/v1/rtb/deals/{id}", limit(perm("rtb:write", h.patchDeal)))
	mux.HandleFunc("DELETE /api/v1/rtb/deals/{id}", limit(perm("rtb:write", h.deleteDeal)))
}

func (h *RtbHTTPHandlers) validateBidRequest(w http.ResponseWriter, r *http.Request) {
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}
	result := openrtb.ValidateBytes(body, h.ExchangeConfig)
	httpresponse.JSON(w, http.StatusOK, result)
}

func (h *RtbHTTPHandlers) integrationProfile(w http.ResponseWriter, r *http.Request) {
	profile := openrtb.Profile()
	resp := RtbIntegrationResponseDTO{IntegrationProfile: profile}
	if h.RuntimeConfig != nil {
		resp.Runtime = RtbRuntimeHintsDTO{
			RtbMode:              h.RuntimeConfig.RtbMode(),
			RtbEnabled:           h.RuntimeConfig.RtbEnabled(),
			RtbExchangeNoBidMode: h.RuntimeConfig.RtbExchangeNoBidMode(),
		}
	}
	if h.PlatformConfig != nil {
		if plat, err := h.PlatformConfig(r.Context()); err == nil {
			domain := plat.TrackingDomain
			resp.Endpoints = RtbEndpointsDTO{
				OpenRTBBidURL:     platformconfig.OpenRTBEndpointTemplate(domain),
				EdgeExposeOpenRTB: plat.EdgeExposeOpenRTB,
				TrackingDomain:    domain,
				EdgePortHint:      ":8180/openrtb/bid",
				TrackerPortHint:   ":8181–8184/openrtb/bid",
			}
		}
	}
	httpresponse.JSON(w, http.StatusOK, resp)
}

func (h *RtbHTTPHandlers) shadowDiff(w http.ResponseWriter, r *http.Request) {
	window := parseWindowQuery(r, time.Hour)
	if h.ShadowDiff != nil {
		httpresponse.JSON(w, http.StatusOK, h.ShadowDiff(window))
		return
	}
	httpresponse.JSON(w, http.StatusOK, RtbShadowDiffSnapshotDTO{Window: window.String(), Source: "unavailable"})
}

type RtbReconcileExportDTO struct {
	Window     string                   `json:"window"`
	RequestID  string                   `json:"request_id,omitempty"`
	Bids       uint64                   `json:"bids"`
	Wins       uint64                   `json:"wins"`
	SpendMicro int64                    `json:"spend_micro"`
	CHSource   string                   `json:"ch_source,omitempty"`
	Shadow     RtbShadowDiffSnapshotDTO `json:"shadow"`
	Ready      bool                     `json:"live_gate_ready"`
	Reasons    []string                 `json:"reasons,omitempty"`
}

type RtbReconcileCHFunc func(ctx context.Context, requestID string, window time.Duration) (bids, wins uint64, spendMicro int64, ok bool)

func (h *RtbHTTPHandlers) reconcileExport(w http.ResponseWriter, r *http.Request) {
	window := parseWindowQuery(r, 24*time.Hour)
	requestID := r.URL.Query().Get("request_id")
	dto := RtbReconcileExportDTO{Window: window.String(), RequestID: requestID}
	if h.ReconcileCH != nil {
		if bids, wins, spend, ok := h.ReconcileCH(r.Context(), requestID, window); ok {
			dto.Bids = bids
			dto.Wins = wins
			dto.SpendMicro = spend
			dto.CHSource = "clickhouse"
		}
	}
	if h.LiveGate != nil {
		gate := h.LiveGate(window)
		dto.Shadow = gate.Shadow
		dto.Ready = gate.Ready
		dto.Reasons = gate.Reasons
		httpresponse.JSON(w, http.StatusOK, dto)
		return
	}
	snap := RtbShadowDiffSnapshotDTO{Window: window.String(), Source: "unavailable"}
	if h.ShadowDiff != nil {
		snap = h.ShadowDiff(window)
	}
	dto.Shadow = snap
	httpresponse.JSON(w, http.StatusOK, dto)
}

func (h *RtbHTTPHandlers) listDeals(w http.ResponseWriter, r *http.Request) {
	rows, err := h.Service.ListRtbDeals(r.Context())
	if err != nil {
		h.writeErr(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, rows)
}

func (h *RtbHTTPHandlers) getDeal(w http.ResponseWriter, r *http.Request) {
	id, err := parsePathInt64(r, "id")
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid deal id")
		return
	}
	row, err := h.Service.GetRtbDeal(r.Context(), id)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, row)
}

func (h *RtbHTTPHandlers) createDeal(w http.ResponseWriter, r *http.Request) {
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}
	spec, err := coldpath.DecodeBody[RtbDealCreateSpec](body)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	row, err := h.Service.CreateRtbDeal(r.Context(), spec)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusCreated, row)
}

func (h *RtbHTTPHandlers) patchDeal(w http.ResponseWriter, r *http.Request) {
	id, err := parsePathInt64(r, "id")
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid deal id")
		return
	}
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}
	spec, err := coldpath.DecodeBody[RtbDealUpdateSpec](body)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	row, err := h.Service.UpdateRtbDeal(r.Context(), id, spec)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, row)
}

func (h *RtbHTTPHandlers) deleteDeal(w http.ResponseWriter, r *http.Request) {
	id, err := parsePathInt64(r, "id")
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid deal id")
		return
	}
	if err := h.Service.DeleteRtbDeal(r.Context(), id); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *RtbHTTPHandlers) writeErr(w http.ResponseWriter, err error) {
	if h.WriteServiceError != nil {
		h.WriteServiceError(w, err)
		return
	}
	httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL", "request failed")
}

func parsePathInt64(r *http.Request, key string) (int64, error) {
	return strconv.ParseInt(r.PathValue(key), 10, 64)
}

func parseWindowQuery(r *http.Request, fallback time.Duration) time.Duration {
	if raw := r.URL.Query().Get("window"); raw != "" {
		if d, err := time.ParseDuration(raw); err == nil && d > 0 {
			return d
		}
	}
	return fallback
}
