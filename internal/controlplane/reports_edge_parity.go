package controlplane

import (
	"context"
	"math"
	"net/http"
	"time"

	"ad-event-processor/pkg/httpresponse"
)

const (
	edgeParityDefaultLookback = 15 * time.Minute
	edgeParityDivergenceAlert = 0.05
)

type EdgeParityReportDTO struct {
	From              string           `json:"from"`
	To                string           `json:"to"`
	EdgeIngress       uint64           `json:"edge_ingress"`
	TrackerEvents     uint64           `json:"tracker_events"`
	DivergencePct     float64          `json:"divergence_pct"`
	Alert             bool             `json:"alert"`
	BlacklistStale    uint64           `json:"blacklist_stale"`
	EdgeBlockedTotal  uint64           `json:"edge_blocked_total"`
	ShardMismatchHint string           `json:"shard_mismatch_hint,omitempty"`
	Freshness         DataFreshnessDTO `json:"freshness"`
}

const edgeParityTrackerEventsQuery = `
SELECT
 (SELECT count() FROM impressions WHERE created_at >= ? AND created_at < ?) +
 (SELECT count() FROM clicks WHERE created_at >= ? AND created_at < ?) AS tracker_events`

func (h *ReportsHTTPHandlers) registerEdgeParityReport(mux *http.ServeMux) {
	limit := h.ApplyRateLimit
	perm := h.RequirePermission
	if perm == nil {
		perm = func(_ string, next http.HandlerFunc) http.HandlerFunc { return next }
	}
	mux.HandleFunc("GET /api/v1/reports/edge-parity", limit(perm("shards:read", h.wrapReport("edge-parity", h.getEdgeParityReport))))
}

func (h *ReportsHTTPHandlers) getEdgeParityReport(w http.ResponseWriter, r *http.Request) {
	now := time.Now().UTC()
	from := now.Add(-edgeParityDefaultLookback)
	to := now
	if fromStr := r.URL.Query().Get("from"); fromStr != "" {
		parsed, err := time.Parse(time.RFC3339, fromStr)
		if err != nil {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid from timestamp")
			return
		}
		from = parsed.UTC()
	}
	if toStr := r.URL.Query().Get("to"); toStr != "" {
		parsed, err := time.Parse(time.RFC3339, toStr)
		if err != nil {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid to timestamp")
			return
		}
		to = parsed.UTC()
	}
	if !to.After(from) {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "to must be after from")
		return
	}
	if to.Sub(from) > edgeParityDefaultLookback {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "edge parity window exceeds 15 minutes")
		return
	}

	var edge EdgeMetricsPanelDTO
	if h.EdgeMetricsReader != nil {
		panel, err := h.EdgeMetricsReader(r.Context())
		if err == nil {
			edge = panel
		}
	}

	var trackerEvents uint64
	if h.ClickHouseQuery != nil {
		clickhouseCtx, cancel := context.WithTimeout(r.Context(), reportClickHouseQueryTimeout)
		defer cancel()
		if err := h.ClickHouseQuery.QueryRow(clickhouseCtx, edgeParityTrackerEventsQuery, from, to, from, to).Scan(&trackerEvents); err != nil {
			h.writeServiceError(w, err)
			return
		}
	}

	edgeIngress := edge.BodyRead
	if edgeIngress == 0 {
		edgeIngress = edge.IngressH1 + edge.IngressH2 + edge.IngressH3
	}
	blockedTotal := sumEdgeBlocked(edge.Blocked)
	divergence := calcEdgeParityDivergence(edgeIngress, trackerEvents)
	alert := divergence > edgeParityDivergenceAlert

	httpresponse.JSON(w, http.StatusOK, EdgeParityReportDTO{
		From:              from.Format(time.RFC3339),
		To:                to.Format(time.RFC3339),
		EdgeIngress:       edgeIngress,
		TrackerEvents:     trackerEvents,
		DivergencePct:     divergence,
		Alert:             alert,
		BlacklistStale:    edge.BlacklistStale,
		EdgeBlockedTotal:  blockedTotal,
		ShardMismatchHint: shardMismatchHint(blockedTotal, edge.BlacklistStale),
		Freshness:         h.reportFreshness(r.Context()),
	})
}

func sumEdgeBlocked(blocked map[string]uint64) uint64 {
	var total uint64
	for _, v := range blocked {
		total += v
	}
	return total
}

func calcEdgeParityDivergence(edgeIngress, trackerEvents uint64) float64 {
	if edgeIngress == 0 {
		if trackerEvents == 0 {
			return 0
		}
		return 1
	}
	return math.Abs(float64(int64(trackerEvents)-int64(edgeIngress))) / float64(edgeIngress)
}

func shardMismatchHint(blockedTotal, blacklistStale uint64) string {
	if blacklistStale > 0 && blockedTotal > 0 {
		return "blacklist_stale_with_edge_blocks"
	}
	if blacklistStale > 0 {
		return "blacklist_stale"
	}
	return ""
}
