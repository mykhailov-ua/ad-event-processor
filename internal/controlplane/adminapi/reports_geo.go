package adminapi

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/bidshard/ad-event-processor/internal/database"
	"github.com/bidshard/ad-event-processor/pkg/coldpath"
	"github.com/bidshard/ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
)

type GeoROIRowDTO struct {
	Country      string               `json:"country"`
	Impressions  int64                `json:"impressions"`
	Clicks       int64                `json:"clicks"`
	Conversions  int64                `json:"conversions"`
	IVTEvents    int64                `json:"ivt_events"`
	IVTRate      float64              `json:"ivt_rate"`
	SpendMicro   int64                `json:"spend_micro"`
	RevenueMicro int64                `json:"revenue_micro"`
	ProfitMicro  int64                `json:"profit_micro"`
	ROIPct       float64              `json:"roi_pct"`
	CTR          float64              `json:"ctr"`
	Compare      *ReportCompareDeltas `json:"compare,omitempty"`
}

type GeoROIReportResponse struct {
	Rows       []GeoROIRowDTO   `json:"rows"`
	Freshness  DataFreshnessDTO `json:"freshness"`
	NextCursor string           `json:"next_cursor,omitempty"`
}

const geoCountryExpr = `coalesce(nullIf(JSONExtractString(payload, 'country'), ''), 'ZZ')`

const geoROIEventQuery = `
SELECT
    campaign_id,
    country,
    sum(impressions) AS impressions,
    sum(clicks) AS clicks,
    sum(conversions) AS conversions,
    sum(ivt_events) AS ivt_events
FROM (
    SELECT
        campaign_id,
        ` + geoCountryExpr + ` AS country,
        count() AS impressions,
        toUInt64(0) AS clicks,
        toUInt64(0) AS conversions,
        toUInt64(0) AS ivt_events
    FROM impressions
    WHERE campaign_id IN (?)
      AND created_at >= ?
      AND created_at < ?
    GROUP BY campaign_id, country
    UNION ALL
    SELECT
        c.campaign_id,
        coalesce(nullIf(JSONExtractString(c.payload, 'country'), ''), 'ZZ') AS country,
        toUInt64(0) AS impressions,
        count() AS clicks,
        toUInt64(0) AS conversions,
        uniqIf(c.click_id, f.click_id != '') AS ivt_events
    FROM clicks AS c
    LEFT JOIN fraud_events AS f
        ON c.click_id = f.click_id AND c.campaign_id = f.campaign_id
    WHERE c.campaign_id IN (?)
      AND c.created_at >= ?
      AND c.created_at < ?
    GROUP BY c.campaign_id, country
    UNION ALL
    SELECT
        campaign_id,
        ` + geoCountryExpr + ` AS country,
        toUInt64(0),
        toUInt64(0),
        count(),
        toUInt64(0)
    FROM conversions
    WHERE campaign_id IN (?)
      AND created_at >= ?
      AND created_at < ?
    GROUP BY campaign_id, country
)
GROUP BY campaign_id, country`

const geoROICampaignSpendQuery = `
SELECT
    campaign_id,
    sum(spend_micro) AS spend_micro,
    sum(revenue_micro) AS revenue_micro
FROM placement_stats_hourly
WHERE campaign_id IN (?)
  AND hour >= ?
  AND hour < ?
GROUP BY campaign_id`

type geoCampaignCountryRow struct {
	CampaignID  string
	Country     string
	Impressions int64
	Clicks      int64
	Conversions int64
	IVTEvents   int64
}

type campaignSpendTotals struct {
	SpendMicro   int64
	RevenueMicro int64
}

func (reports *ReportsHTTPHandlers) registerGeoROI(mux *http.ServeMux) {
	limit := reports.ApplyRateLimit
	perm := reports.RequirePermission
	mux.HandleFunc("GET /api/v1/reports/geo-roi", limit(perm("campaigns:read", reports.getGeoROIReport)))
}

func (reports *ReportsHTTPHandlers) getGeoROIReport(w http.ResponseWriter, r *http.Request) {
	customerID, ok := reports.resolveReportCustomerID(w, r)
	if !ok {
		return
	}
	if reports.CHQuery == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "CLICKHOUSE_UNAVAILABLE", "clickhouse not configured")
		return
	}
	from, to, err := parseReportRange(r)
	if err != nil {
		reports.writeServiceError(w, err)
		return
	}
	page, err := coldpath.ParseCursorPagination(r, 50, 1000)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid cursor")
		return
	}
	campaignIDs, err := listCustomerCampaignIDs(r.Context(), reports.Pool, customerID)
	if err != nil {
		reports.writeServiceError(w, err)
		return
	}
	if len(campaignIDs) == 0 {
		httpresponse.JSON(w, http.StatusOK, GeoROIReportResponse{
			Rows:      []GeoROIRowDTO{},
			Freshness: reports.reportFreshness(r.Context()),
		})
		return
	}
	chCtx, cancel := context.WithTimeout(r.Context(), reportCHQueryTimeout)
	defer cancel()
	rows, total, err := queryGeoROIRows(chCtx, reports.CHQuery, campaignIDs, from, to, page.Limit, page.Offset)
	if err != nil {
		reports.writeServiceError(w, err)
		return
	}
	if parseComparePrevious(r) {
		prevFrom, prevTo := previousReportRange(from, to)
		prevRows, _, perr := queryGeoROIRows(chCtx, reports.CHQuery, campaignIDs, prevFrom, prevTo, page.Limit, page.Offset)
		if perr != nil {
			reports.writeServiceError(w, perr)
			return
		}
		attachGeoCompareDeltas(rows, prevRows)
	}
	var nextCursor string
	if int64(page.Offset)+int64(len(rows)) < total {
		nextCursor = coldpath.EncodeCursor(page.Offset + page.Limit)
	}
	httpresponse.JSON(w, http.StatusOK, GeoROIReportResponse{
		Rows:       rows,
		Freshness:  reports.reportFreshness(r.Context()),
		NextCursor: nextCursor,
	})
}

func queryGeoROIRows(
	ctx context.Context,
	chQuery *database.CHQuery,
	campaignIDs []uuid.UUID,
	from, to time.Time,
	limit, offset int,
) ([]GeoROIRowDTO, int64, error) {
	if chQuery == nil || len(campaignIDs) == 0 {
		return nil, 0, nil
	}
	rows, err := chQuery.Query(ctx, geoROIEventQuery,
		campaignIDs, from, to,
		campaignIDs, from, to,
		campaignIDs, from, to,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("geo roi event query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	eventRows := make([]geoCampaignCountryRow, 0, 256)
	campaignClicks := make(map[string]int64)
	campaignImpressions := make(map[string]int64)
	for rows.Next() {
		var row geoCampaignCountryRow
		var campaignID uuid.UUID
		if err := rows.Scan(
			&campaignID, &row.Country,
			&row.Impressions, &row.Clicks, &row.Conversions, &row.IVTEvents,
		); err != nil {
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
	spendRows, err := chQuery.Query(ctx, geoROICampaignSpendQuery, campaignIDs, from, to)
	if err != nil {
		return nil, 0, fmt.Errorf("geo roi spend query: %w", err)
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

	merged := mergeGeoRowsByCountry(eventRows, campaignClicks, campaignImpressions, spendByCampaign)
	if offset >= len(merged) {
		return []GeoROIRowDTO{}, int64(len(merged)), nil
	}
	end := offset + limit
	if end > len(merged) {
		end = len(merged)
	}
	return merged[offset:end], int64(len(merged)), nil
}

func mergeGeoRowsByCountry(
	rows []geoCampaignCountryRow,
	campaignClicks, campaignImpressions map[string]int64,
	spendByCampaign map[string]campaignSpendTotals,
) []GeoROIRowDTO {
	type agg struct {
		impressions, clicks, conversions, ivt, spend, revenue int64
	}
	byCountry := make(map[string]*agg)
	for _, row := range rows {
		a := byCountry[row.Country]
		if a == nil {
			a = &agg{}
			byCountry[row.Country] = a
		}
		a.impressions += row.Impressions
		a.clicks += row.Clicks
		a.conversions += row.Conversions
		a.ivt += row.IVTEvents

		totals := spendByCampaign[row.CampaignID]
		if totals.SpendMicro == 0 && totals.RevenueMicro == 0 {
			continue
		}
		share := allocateGeoShare(row, campaignClicks[row.CampaignID], campaignImpressions[row.CampaignID])
		if share <= 0 {
			continue
		}
		a.spend += int64(float64(totals.SpendMicro) * share)
		a.revenue += int64(float64(totals.RevenueMicro) * share)
	}

	out := make([]GeoROIRowDTO, 0, len(byCountry))
	for country, a := range byCountry {
		profit := a.revenue - a.spend
		out = append(out, GeoROIRowDTO{
			Country:      country,
			Impressions:  a.impressions,
			Clicks:       a.clicks,
			Conversions:  a.conversions,
			IVTEvents:    a.ivt,
			IVTRate:      calcIVTRate(a.ivt, a.clicks),
			SpendMicro:   a.spend,
			RevenueMicro: a.revenue,
			ProfitMicro:  profit,
			ROIPct:       calcROIPct(profit, a.spend),
			CTR:          calcCTR(a.clicks, a.impressions),
		})
	}
	sortGeoROIRows(out)
	return out
}

func allocateGeoShare(row geoCampaignCountryRow, campaignClicks, campaignImpressions int64) float64 {
	if campaignClicks > 0 && row.Clicks > 0 {
		return float64(row.Clicks) / float64(campaignClicks)
	}
	if campaignImpressions > 0 && row.Impressions > 0 {
		return float64(row.Impressions) / float64(campaignImpressions)
	}
	return 0
}

func sortGeoROIRows(rows []GeoROIRowDTO) {
	for i := range rows {
		for j := i + 1; j < len(rows); j++ {
			if rows[j].SpendMicro > rows[i].SpendMicro ||
				(rows[j].SpendMicro == rows[i].SpendMicro && rows[j].IVTEvents > rows[i].IVTEvents) {
				rows[i], rows[j] = rows[j], rows[i]
			}
		}
	}
}
