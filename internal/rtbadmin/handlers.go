package rtbadmin

import (
	"net/http"
	"strconv"
	"time"

	"ad-event-processor/internal/licensingadmin"
	"ad-event-processor/internal/openrtb"
	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"
	"ad-event-processor/pkg/platformconfig"
)

type HTTPHandlers struct {
	Service               DealService
	ApplyRateLimit        func(http.HandlerFunc) http.HandlerFunc
	RequirePermission     func(string, http.HandlerFunc) http.HandlerFunc
	WriteServiceError     func(http.ResponseWriter, error)
	RequireLicenseFeature licensingadmin.FeatureChecker
	ExchangeConfig        openrtb.ExchangeConfig
	ShadowDiff            func(time.Duration) ShadowDiffSnapshotDTO
	LiveGate              func(time.Duration) LiveGateDTO
	ReconcileCH           ReconcileCHFunc
	RuntimeConfig         RuntimeConfigReader
	PlatformConfig        PlatformConfigReader
}

type RuntimeHintsDTO struct {
	RtbMode              string `json:"rtb_mode"`
	RtbEnabled           bool   `json:"rtb_enabled"`
	RtbExchangeNoBidMode string `json:"rtb_exchange_no_bid_mode,omitempty"`
}

type EndpointsDTO struct {
	OpenRTBBidURL     string `json:"openrtb_bid_url"`
	EdgeExposeOpenRTB bool   `json:"edge_expose_openrtb"`
	TrackingDomain    string `json:"tracking_domain,omitempty"`
	EdgePortHint      string `json:"edge_port_hint,omitempty"`
	TrackerPortHint   string `json:"tracker_port_hint,omitempty"`
}

type IntegrationResponseDTO struct {
	openrtb.IntegrationProfile
	Runtime   RuntimeHintsDTO `json:"runtime"`
	Endpoints EndpointsDTO    `json:"endpoints"`
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
	gate := func(next http.HandlerFunc) http.HandlerFunc {
		return limit(perm("rtb:read", func(w http.ResponseWriter, r *http.Request) {
			if !licensingadmin.RequireLicenseFeature(w, h.RequireLicenseFeature, "openrtb") {
				return
			}
			next(w, r)
		}))
	}
	gateWrite := func(next http.HandlerFunc) http.HandlerFunc {
		return limit(perm("rtb:write", func(w http.ResponseWriter, r *http.Request) {
			if !licensingadmin.RequireLicenseFeature(w, h.RequireLicenseFeature, "openrtb") {
				return
			}
			next(w, r)
		}))
	}
	mux.HandleFunc("POST /api/v1/rtb/validate-bid-request", gate(h.validateBidRequest))
	mux.HandleFunc("GET /api/v1/rtb/integration-profile", gate(h.integrationProfile))
	mux.HandleFunc("GET /api/v1/rtb/shadow-diff", gate(h.shadowDiff))
	mux.HandleFunc("GET /api/v1/rtb/reconcile/export", gate(h.reconcileExport))
	mux.HandleFunc("GET /api/v1/rtb/deals", gate(h.listDeals))
	mux.HandleFunc("GET /api/v1/rtb/deals/{id}", gate(h.getDeal))
	mux.HandleFunc("POST /api/v1/rtb/deals", gateWrite(h.createDeal))
	mux.HandleFunc("PATCH /api/v1/rtb/deals/{id}", gateWrite(h.patchDeal))
	mux.HandleFunc("DELETE /api/v1/rtb/deals/{id}", gateWrite(h.deleteDeal))
}

func (h *HTTPHandlers) validateBidRequest(w http.ResponseWriter, r *http.Request) {
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}
	result := openrtb.ValidateBytes(body, h.ExchangeConfig)
	httpresponse.JSON(w, http.StatusOK, result)
}

func (h *HTTPHandlers) integrationProfile(w http.ResponseWriter, r *http.Request) {
	profile := openrtb.Profile()
	resp := IntegrationResponseDTO{IntegrationProfile: profile}
	if h.RuntimeConfig != nil {
		resp.Runtime = RuntimeHintsDTO{
			RtbMode:              h.RuntimeConfig.RtbMode(),
			RtbEnabled:           h.RuntimeConfig.RtbEnabled(),
			RtbExchangeNoBidMode: h.RuntimeConfig.RtbExchangeNoBidMode(),
		}
	}
	if h.PlatformConfig != nil {
		if plat, err := h.PlatformConfig(r.Context()); err == nil {
			domain := plat.TrackingDomain
			resp.Endpoints = EndpointsDTO{
				OpenRTBBidURL:     platformconfig.OpenRTBEndpointTemplate(domain),
				EdgeExposeOpenRTB: plat.EdgeExposeOpenRTB,
				TrackingDomain:    domain,
				EdgePortHint:      ":8180/openrtb/bid",
				TrackerPortHint:   ":8181-8184/openrtb/bid",
			}
		}
	}
	httpresponse.JSON(w, http.StatusOK, resp)
}

func (h *HTTPHandlers) shadowDiff(w http.ResponseWriter, r *http.Request) {
	window := parseWindowQuery(r, time.Hour)
	if h.ShadowDiff != nil {
		httpresponse.JSON(w, http.StatusOK, h.ShadowDiff(window))
		return
	}
	httpresponse.JSON(w, http.StatusOK, ShadowDiffSnapshotDTO{Window: window.String(), Source: "unavailable"})
}

type ReconcileExportDTO struct {
	Window     string                `json:"window"`
	RequestID  string                `json:"request_id,omitempty"`
	Bids       uint64                `json:"bids"`
	Wins       uint64                `json:"wins"`
	SpendMicro int64                 `json:"spend_micro"`
	CHSource   string                `json:"ch_source,omitempty"`
	Shadow     ShadowDiffSnapshotDTO `json:"shadow"`
	Ready      bool                  `json:"live_gate_ready"`
	Reasons    []string              `json:"reasons,omitempty"`
}

func (h *HTTPHandlers) reconcileExport(w http.ResponseWriter, r *http.Request) {
	window := parseWindowQuery(r, 24*time.Hour)
	requestID := r.URL.Query().Get("request_id")
	dto := ReconcileExportDTO{Window: window.String(), RequestID: requestID}
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
	snap := ShadowDiffSnapshotDTO{Window: window.String(), Source: "unavailable"}
	if h.ShadowDiff != nil {
		snap = h.ShadowDiff(window)
	}
	dto.Shadow = snap
	httpresponse.JSON(w, http.StatusOK, dto)
}

func (h *HTTPHandlers) listDeals(w http.ResponseWriter, r *http.Request) {
	rows, err := h.Service.ListRtbDeals(r.Context())
	if err != nil {
		h.writeErr(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, rows)
}

func (h *HTTPHandlers) getDeal(w http.ResponseWriter, r *http.Request) {
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

func (h *HTTPHandlers) createDeal(w http.ResponseWriter, r *http.Request) {
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}
	spec, err := coldpath.DecodeBody[DealCreateSpec](body)
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

func (h *HTTPHandlers) patchDeal(w http.ResponseWriter, r *http.Request) {
	id, err := parsePathInt64(r, "id")
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid deal id")
		return
	}
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}
	spec, err := coldpath.DecodeBody[DealUpdateSpec](body)
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

func (h *HTTPHandlers) deleteDeal(w http.ResponseWriter, r *http.Request) {
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

func (h *HTTPHandlers) writeErr(w http.ResponseWriter, err error) {
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
