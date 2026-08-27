package controlplane

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"time"

	"ad-event-processor/internal/database"
	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
)

type TrafficSourceRowDTO struct {
	Channel      string               `json:"channel"`
	Impressions  int64                `json:"impressions"`
	Clicks       int64                `json:"clicks"`
	Conversions  int64                `json:"conversions"`
	SpendMicro   int64                `json:"spend_micro"`
	RevenueMicro int64                `json:"revenue_micro"`
	ProfitMicro  int64                `json:"profit_micro"`
	ROIPct       float64              `json:"roi_pct"`
	CTR          float64              `json:"ctr"`
	Compare      *ReportCompareDeltas `json:"compare,omitempty"`
}

type TrafficSourcesReportResponse struct {
	Rows       []TrafficSourceRowDTO `json:"rows"`
	Freshness  DataFreshnessDTO      `json:"freshness"`
	NextCursor string                `json:"next_cursor,omitempty"`
}

const trafficChannelExpr = `
multiIf(
 JSONExtractString(payload, 'gclid') != '', 'paid_search',
 JSONExtractString(payload, 'fbclid') != '' OR JSONExtractString(payload, 'ttclid') != '', 'paid_social',
 lower(JSONExtractString(payload, 'sub1')) = 'organic' OR lower(JSONExtractString(payload, 'sub2')) = 'organic', 'organic',
 lower(JSONExtractString(payload, 'sub1')) = 'email' OR lower(JSONExtractString(payload, 'sub2')) = 'email', 'email',
 lower(JSONExtractString(payload, 'sub1')) = 'affiliate' OR lower(JSONExtractString(payload, 'sub2')) = 'affiliate', 'affiliate',
 JSONExtractString(payload, 'sub1') != '' OR JSONExtractString(payload, 'sub2') != '', 'custom',
 'direct'
)`

const trafficEventQuery = `
SELECT
 campaign_id,
 channel,
 sum(impressions) AS impressions,
 sum(clicks) AS clicks,
 sum(conversions) AS conversions
FROM (
 SELECT
 campaign_id,
 ` + trafficChannelExpr + ` AS channel,
 count() AS impressions,
 toUInt64(0) AS clicks,
 toUInt64(0) AS conversions
 FROM impressions
 WHERE campaign_id IN (?)
 AND created_at >= ?
 AND created_at < ?
 GROUP BY campaign_id, channel
 UNION ALL
 SELECT
 campaign_id,
 ` + trafficChannelExpr + ` AS channel,
 toUInt64(0),
 count(),
 toUInt64(0)
 FROM clicks
 WHERE campaign_id IN (?)
 AND created_at >= ?
 AND created_at < ?
 GROUP BY campaign_id, channel
 UNION ALL
 SELECT
 campaign_id,
 ` + trafficChannelExpr + ` AS channel,
 toUInt64(0),
 toUInt64(0),
 count()
 FROM conversions
 WHERE campaign_id IN (?)
 AND created_at >= ?
 AND created_at < ?
 GROUP BY campaign_id, channel
)
GROUP BY campaign_id, channel`

type trafficCampaignChannelRow struct {
	CampaignID  string
	Channel     string
	Impressions int64
	Clicks      int64
	Conversions int64
}

func (h *ReportsHTTPHandlers) registerTrafficSources(mux *http.ServeMux) {
	limit := h.ApplyRateLimit
	perm := h.RequirePermission
	mux.HandleFunc("GET /api/v1/reports/traffic-sources", limit(perm("campaigns:read", h.wrapReport("traffic-sources", h.getTrafficSourcesReport))))
}

func (h *ReportsHTTPHandlers) getTrafficSourcesReport(w http.ResponseWriter, r *http.Request) {
	customerID, ok := h.resolveReportCustomerID(w, r)
	if !ok {
		return
	}
	if h.ClickHouseQuery == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "CLICKHOUSE_UNAVAILABLE", "clickhouse not configured")
		return
	}
	from, to, err := parseReportRange(r)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	page, err := coldpath.ParseCursorPagination(r, 50, 1000)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid cursor")
		return
	}
	campaignIDs, err := listCustomerCampaignIDs(r.Context(), h.Pool, customerID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	if len(campaignIDs) == 0 {
		httpresponse.JSON(w, http.StatusOK, TrafficSourcesReportResponse{
			Rows:      []TrafficSourceRowDTO{},
			Freshness: h.reportFreshness(r.Context()),
		})
		return
	}
	clickhouseCtx, cancel := context.WithTimeout(r.Context(), reportClickHouseQueryTimeout)
	defer cancel()
	rows, total, err := queryTrafficSourceRows(clickhouseCtx, h.ClickHouseQuery, campaignIDs, from, to, page.Limit, page.Offset)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	if parseComparePrevious(r) {
		prevFrom, prevTo := previousReportRange(from, to)
		prevRows, _, perr := queryTrafficSourceRows(clickhouseCtx, h.ClickHouseQuery, campaignIDs, prevFrom, prevTo, page.Limit, page.Offset)
		if perr != nil {
			h.writeServiceError(w, perr)
			return
		}
		attachTrafficCompareDeltas(rows, prevRows)
	}
	var nextCursor string
	if int64(page.Offset)+int64(len(rows)) < total {
		nextCursor = coldpath.EncodeCursor(page.Offset + page.Limit)
	}
	httpresponse.JSON(w, http.StatusOK, TrafficSourcesReportResponse{
		Rows:       rows,
		Freshness:  h.reportFreshness(r.Context()),
		NextCursor: nextCursor,
	})
}

func queryTrafficSourceRows(
	ctx context.Context,
	clickhouseQuery *database.CHQuery,
	campaignIDs []uuid.UUID,
	from, to time.Time,
	limit, offset int,
) ([]TrafficSourceRowDTO, int64, error) {
	if clickhouseQuery == nil || len(campaignIDs) == 0 {
		return nil, 0, nil
	}
	rows, err := clickhouseQuery.Query(ctx, trafficEventQuery,
		campaignIDs, from, to,
		campaignIDs, from, to,
		campaignIDs, from, to,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("traffic sources query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	eventRows := make([]trafficCampaignChannelRow, 0, 256)
	campaignClicks := make(map[string]int64)
	campaignImpressions := make(map[string]int64)
	for rows.Next() {
		var row trafficCampaignChannelRow
		var campaignID uuid.UUID
		if err := rows.Scan(&campaignID, &row.Channel, &row.Impressions, &row.Clicks, &row.Conversions); err != nil {
			return nil, 0, err
		}
		row.CampaignID = campaignID.String()
		eventRows = append(eventRows, row)
		campaignClicks[row.CampaignID] += row.Clicks
		campaignImpressions[row.CampaignID] += row.Impressions
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	spendByCampaign := make(map[string]campaignSpendTotals, len(campaignIDs))
	spendRows, err := clickhouseQuery.Query(ctx, geoROICampaignSpendQuery, campaignIDs, from, to)
	if err != nil {
		return nil, 0, fmt.Errorf("traffic spend query: %w", err)
	}
	defer func() { _ = spendRows.Close() }()
	for spendRows.Next() {
		var campaignID uuid.UUID
		var totals campaignSpendTotals
		if err := spendRows.Scan(&campaignID, &totals.SpendMicro, &totals.RevenueMicro); err != nil {
			return nil, 0, err
		}
		spendByCampaign[campaignID.String()] = totals
	}
	if err := spendRows.Err(); err != nil {
		return nil, 0, err
	}

	merged := mergeTrafficRowsByChannel(eventRows, campaignClicks, campaignImpressions, spendByCampaign)
	if offset >= len(merged) {
		return []TrafficSourceRowDTO{}, int64(len(merged)), nil
	}
	end := offset + limit
	if end > len(merged) {
		end = len(merged)
	}
	return merged[offset:end], int64(len(merged)), nil
}

func mergeTrafficRowsByChannel(
	rows []trafficCampaignChannelRow,
	campaignClicks, campaignImpressions map[string]int64,
	spendByCampaign map[string]campaignSpendTotals,
) []TrafficSourceRowDTO {
	type agg struct {
		impressions, clicks, conversions, spend, revenue int64
	}
	byChannel := make(map[string]*agg)
	for _, row := range rows {
		a := byChannel[row.Channel]
		if a == nil {
			a = &agg{}
			byChannel[row.Channel] = a
		}
		a.impressions += row.Impressions
		a.clicks += row.Clicks
		a.conversions += row.Conversions

		totals := spendByCampaign[row.CampaignID]
		if totals.SpendMicro == 0 && totals.RevenueMicro == 0 {
			continue
		}
		share := allocateTrafficShare(row, campaignClicks[row.CampaignID], campaignImpressions[row.CampaignID])
		if share <= 0 {
			continue
		}
		a.spend += int64(float64(totals.SpendMicro) * share)
		a.revenue += int64(float64(totals.RevenueMicro) * share)
	}

	out := make([]TrafficSourceRowDTO, 0, len(byChannel))
	for channel, a := range byChannel {
		profit := a.revenue - a.spend
		out = append(out, TrafficSourceRowDTO{
			Channel:      channel,
			Impressions:  a.impressions,
			Clicks:       a.clicks,
			Conversions:  a.conversions,
			SpendMicro:   a.spend,
			RevenueMicro: a.revenue,
			ProfitMicro:  profit,
			ROIPct:       calcROIPct(profit, a.spend),
			CTR:          calcCTR(a.clicks, a.impressions),
		})
	}
	sortTrafficRows(out)
	return out
}

func allocateTrafficShare(row trafficCampaignChannelRow, campaignClicks, campaignImpressions int64) float64 {
	if campaignClicks > 0 && row.Clicks > 0 {
		return float64(row.Clicks) / float64(campaignClicks)
	}
	if campaignImpressions > 0 && row.Impressions > 0 {
		return float64(row.Impressions) / float64(campaignImpressions)
	}
	return 0
}

func sortTrafficRows(rows []TrafficSourceRowDTO) {
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].SpendMicro != rows[j].SpendMicro {
			return rows[i].SpendMicro > rows[j].SpendMicro
		}
		return rows[i].Clicks > rows[j].Clicks
	})
}
